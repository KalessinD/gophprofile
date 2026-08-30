package config

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"
)

type (
	// workerConfigJSON is used only for JSON file unmarshaling
	workerConfigJSON struct {
		S3Address                string `json:"s3_address,omitempty"`
		S3UseSSL                 bool   `json:"s3_use_ssl,omitempty"`
		S3AccessKey              string `json:"s3_access_key,omitempty"`
		S3Secretkey              string `json:"s3_secretkey,omitempty"`
		S3Bucket                 string `json:"s3_bucket,omitempty"`
		DatabaseDSN              string `json:"database_dsn,omitempty"`
		KafkaBrokers             string `json:"kafka_brokers,omitempty"`
		KafkaTopic               string `json:"kafka_topic,omitempty"`
		KafkaGroupID             string `json:"kafka_group_id,omitempty"`
		LoggerType               string `json:"logger_type"`
		OTELExporterOTLPEndpoint string `json:"otel_exporter_otlp_endpoint,omitempty"`
	}

	// Worker configuration struct
	WorkerConfig struct {
		LoggerType               string
		PsqlDSN                  string
		CompressionThreshold     int
		S3                       *S3
		Kafka                    *Kafka
		GracefullShutdownTimeout time.Duration
		OTELExporterOTLPEndpoint string
	}
)

// GetDefaultWorkerConfig returns the default worker configuration
func GetDefaultWorkerConfig() *WorkerConfig {
	return &WorkerConfig{
		LoggerType:               DefaultLoggerType,
		PsqlDSN:                  DefaultPsqlDSN,
		S3:                       getDefaultS3(),
		Kafka:                    getDefaultKafka(),
		GracefullShutdownTimeout: DefaultGracefullShutdownTimeout,
		OTELExporterOTLPEndpoint: DefaultOTELExporterOTLPEndpoint,
	}
}

// Validate validates the worker configuration and returns error if its occurred
func (c *WorkerConfig) Validate() error {
	if c.LoggerType != "" {
		if _, ok := validLogTypes[c.LoggerType]; !ok {
			return errors.New("invalid logger type. Supported types are zap and slog")
		}
	}

	return nil
}

// UpdateFromEnvironment updates settings from ENV variables
func (c *WorkerConfig) UpdateFromEnvironment() error {
	c.PsqlDSN = GetEnvOrFallback("DATABASE_DSN", c.PsqlDSN)
	c.S3.ListenAddr = GetEnvOrFallback("S3_ENDPOINT", c.S3.ListenAddr)
	c.S3.UseSSL = GetEnvOrFallback("S3_USE_SSL", c.S3.UseSSL)
	c.S3.AccessKey = GetEnvOrFallback("S3_ACCESS_KEY", c.S3.AccessKey)
	c.S3.SecretKey = GetEnvOrFallback("S3_SECRET_KEY", c.S3.SecretKey)
	c.S3.Bucket = GetEnvOrFallback("S3_BUCKET", c.S3.Bucket)
	c.Kafka.Brokers = GetEnvOrFallback("KAFKA_BROKERS", c.Kafka.Brokers)
	c.Kafka.Topic = GetEnvOrFallback("KAFKA_TOPIC", c.Kafka.Topic)
	c.Kafka.GroupID = GetEnvOrFallback("KAFKA_GROUP_ID", c.Kafka.GroupID)
	c.LoggerType = GetEnvOrFallback("LOGGER_TYPE", c.LoggerType)
	c.OTELExporterOTLPEndpoint = GetEnvOrFallback("OTEL_EXPORTER_OTLP_ENDPOINT", c.OTELExporterOTLPEndpoint)

	return nil
}

// UpdateFromCLIArgs updates settings from CLI arguments
func (c *WorkerConfig) UpdateFromCLIArgs(flagSet *flag.FlagSet, args []string) error {
	flagSet.BoolVar(&c.S3.UseSSL, "s3-use-ssl", c.S3.UseSSL, "turns on usage of SSL for S3 connections")

	flagSet.StringVar(&c.PsqlDSN, "d", c.PsqlDSN, "SQL database DSN string")
	flagSet.StringVar(&c.S3.ListenAddr, "g", c.S3.ListenAddr, "S3 listen address")
	flagSet.StringVar(&c.S3.AccessKey, "s3-access-key", c.S3.AccessKey, "S3 access key")
	flagSet.StringVar(&c.S3.SecretKey, "s3-secret-key", c.S3.SecretKey, "S3 secret key")
	flagSet.StringVar(&c.S3.Bucket, "s3-bucket", c.S3.Bucket, "S3 bucket")
	flagSet.StringVar(&c.Kafka.Brokers, "kafka-brokers", c.Kafka.Brokers, "Kafka broker addresses (comma separated)")
	flagSet.StringVar(&c.Kafka.Topic, "kafka-topic", c.Kafka.Topic, "Kafka topic for avatar processing")
	flagSet.StringVar(&c.Kafka.GroupID, "kafka-group-id", c.Kafka.GroupID, "Kafka consumer group ID")
	flagSet.StringVar(&c.LoggerType, "logger-type", c.LoggerType, "logger type: zap, slog (default)")
	flagSet.StringVar(&c.OTELExporterOTLPEndpoint, "otel-endpoint", c.OTELExporterOTLPEndpoint, "OTLP exporter endpoint (e.g., jaeger:4317)")

	var dummy string
	flagSet.StringVar(&dummy, "c", "", "path to configuration file")
	flagSet.StringVar(&dummy, "config", "", "path to the configurtion file")

	if err := flagSet.Parse(args); err != nil {
		return err
	}

	return nil
}

// UpdateFromFile checks the config file, and if it exists then tries to parse it
func (c *WorkerConfig) UpdateFromFile(configFile string) error {
	fileHandler, err := os.OpenFile(configFile, os.O_RDONLY, 0o600)
	if err != nil {
		return fmt.Errorf("can't read the configuration file %s: %w", configFile, err)
	}
	defer fileHandler.Close()

	data, err := io.ReadAll(fileHandler)
	if err != nil {
		return err
	}

	if len(data) == 0 {
		return nil
	}

	var jsonCfg workerConfigJSON
	if err := json.Unmarshal(data, &jsonCfg); err != nil {
		return err
	}

	if jsonCfg.DatabaseDSN != "" {
		c.PsqlDSN = jsonCfg.DatabaseDSN
	}

	if jsonCfg.S3Address != "" {
		c.S3.ListenAddr = jsonCfg.S3Address
	}

	if jsonCfg.S3AccessKey != "" {
		c.S3.AccessKey = jsonCfg.S3AccessKey
	}

	if jsonCfg.S3Secretkey != "" {
		c.S3.SecretKey = jsonCfg.S3Secretkey
	}

	if jsonCfg.S3Bucket != "" {
		c.S3.Bucket = jsonCfg.S3Bucket
	}

	if jsonCfg.S3UseSSL != c.S3.UseSSL {
		c.S3.UseSSL = jsonCfg.S3UseSSL
	}

	if jsonCfg.KafkaBrokers != "" {
		c.Kafka.Brokers = jsonCfg.KafkaBrokers
	}

	if jsonCfg.KafkaTopic != "" {
		c.Kafka.Topic = jsonCfg.KafkaTopic
	}

	if jsonCfg.KafkaGroupID != "" {
		c.Kafka.GroupID = jsonCfg.KafkaGroupID
	}

	if jsonCfg.LoggerType != "" {
		c.LoggerType = jsonCfg.LoggerType
	}

	if jsonCfg.OTELExporterOTLPEndpoint != "" {
		c.OTELExporterOTLPEndpoint = jsonCfg.OTELExporterOTLPEndpoint
	}

	return nil
}

// Returns the instance of worker configuration struct.
//
// Fills it's fields by using CLI arguments and environments.
// ENV or CLI argument or the default values
func NewWorkerConfig(flagSet *flag.FlagSet, args []string) (*WorkerConfig, error) {
	cfg := GetDefaultWorkerConfig()
	configFile := GetEnvOrFallback("CONFIG", ParseConfigPath(args))

	if configFile != "" {
		if err := cfg.UpdateFromFile(configFile); err != nil {
			return nil, err
		}
	}

	if err := cfg.UpdateFromEnvironment(); err != nil {
		return nil, err
	}

	if err := cfg.UpdateFromCLIArgs(flagSet, args); err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}
