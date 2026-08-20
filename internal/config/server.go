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

const (
	DefaultListenAddr               string        = ":8080"
	DefaultS3ListenAddr             string        = ":9000"
	DefaultProcessingTimeout        time.Duration = 60 * time.Second
	DefaultReadTimeout              time.Duration = 5 * time.Second
	DefaultReadHeaderTimeout        time.Duration = 5 * time.Second
	DefaultWriteTimeout             time.Duration = 10 * time.Second
	DefaultIdleTimeout              time.Duration = 30 * time.Second
	DefaultGracefullShutdownTimeout time.Duration = 5 * time.Second
	DefaultPsqlDSN                  string        = ""
	DefaultCompressionThreshold     int           = 1024
	DefaultApplyMigrations          bool          = false
	DefaultS3UseSSL                 bool          = false
	DefaultS3AccessKey              string        = ""
	DefaultS3Secretkey              string        = ""
	DefaultS3Bucket                 string        = "gophprofile"
	DefaultKafkaBrokers             string        = ""
	DefaultKafkaTopic               string        = "avatar-processing"
)

type (
	// serverConfigJSON is used only for JSON file unmarshaling
	serverConfigJSON struct {
		Address         string `json:"address,omitempty"`
		S3Address       string `json:"s3_address,omitempty"`
		S3UseSSL        bool   `json:"s3_use_ssl,omitempty"`
		S3AccessKey     string `json:"s3_access_key,omitempty"`
		S3Secretkey     string `json:"s3_secretkey,omitempty"`
		S3Bucket        string `json:"s3_bucket,omitempty"`
		DatabaseDSN     string `json:"database_dsn,omitempty"`
		KafkaBrokers    string `json:"kafka_brokers,omitempty"`
		KafkaTopic      string `json:"kafka_topic,omitempty"`
		ApplyMigrations bool   `json:"apply_migrations"`

		TLS *struct {
			CertFile string `json:"cert_file,omitempty"`
			KeyFile  string `json:"key_file,omitempty"`
		} `json:"tls,omitempty"`
	}

	// Server configuration struct
	ServerConfig struct {
		ListenAddr               string
		S3ListenAddr             string
		ProcessingTimeout        time.Duration
		ReadTimeout              time.Duration
		ReadHeaderTimeout        time.Duration
		WriteTimeout             time.Duration
		IdleTimeout              time.Duration
		GracefullShutdownTimeout time.Duration
		PsqlDSN                  string
		TLSConfig                *TLSConfig
		CompressionThreshold     int
		S3UseSSL                 bool
		S3AccessKey              string
		S3SecretKey              string
		S3Bucket                 string
		ApplyMigrations          bool
		KafkaBrokers             string
		KafkaTopic               string
	}

	// ServerConfigurator is an interface to be implemented by server configuration object
	ServerConfigurator interface {
		UpdateFromEnvironment() error
		UpdateFromCLIArgs(flagSet *flag.FlagSet, args []string) error
		UpdateFromFile(configFile string) error
		Validate() error
	}

	// TLSConfig contains paths to TLS certificates.
	TLSConfig struct {
		CertFile string
		KeyFile  string
	}
)

// GetDefaultServerConfig returns the default server configuration
func GetDefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		ListenAddr:               DefaultListenAddr,
		S3ListenAddr:             DefaultS3ListenAddr,
		ProcessingTimeout:        DefaultProcessingTimeout,
		ReadTimeout:              DefaultReadTimeout,
		ReadHeaderTimeout:        DefaultReadHeaderTimeout,
		WriteTimeout:             DefaultWriteTimeout,
		IdleTimeout:              DefaultIdleTimeout,
		GracefullShutdownTimeout: DefaultGracefullShutdownTimeout,
		PsqlDSN:                  DefaultPsqlDSN,
		CompressionThreshold:     DefaultCompressionThreshold,
		ApplyMigrations:          DefaultApplyMigrations,
		S3UseSSL:                 DefaultS3UseSSL,
		S3AccessKey:              DefaultS3AccessKey,
		S3SecretKey:              DefaultS3Secretkey,
		S3Bucket:                 DefaultS3Bucket,
		KafkaBrokers:             DefaultKafkaBrokers,
		KafkaTopic:               DefaultKafkaTopic,
	}
}

// Validate validates the server configuration and returns error if its occurred
func (c *ServerConfig) Validate() error {
	if err := ValidateAddr(c.ListenAddr); err != nil {
		return err
	}
	if c.S3ListenAddr != "" {
		if err := ValidateAddr(c.S3ListenAddr); err != nil {
			return err
		}
	}

	if c.TLSConfig != nil {
		if (c.TLSConfig.CertFile != "" && c.TLSConfig.KeyFile == "") || (c.TLSConfig.CertFile == "" && c.TLSConfig.KeyFile != "") {
			return errors.New("both TLS certificate and key files must be provided")
		}
	}

	return nil
}

// UpdateFromEnvironment updates settings from ENV variables
func (c *ServerConfig) UpdateFromEnvironment() error {
	c.ListenAddr = GetEnvOrFallback("ADDRESS", c.ListenAddr)
	c.PsqlDSN = GetEnvOrFallback("DATABASE_DSN", c.PsqlDSN)
	c.S3ListenAddr = GetEnvOrFallback("S3_ENDPOINT", c.S3ListenAddr)
	c.S3UseSSL = GetEnvOrFallback("S3_USE_SSL", c.S3UseSSL)
	c.S3AccessKey = GetEnvOrFallback("S3_ACCESS_KEY", c.S3AccessKey)
	c.S3SecretKey = GetEnvOrFallback("S3_SECRET_KEY", c.S3SecretKey)
	c.S3Bucket = GetEnvOrFallback("S3_BUCKET", c.S3Bucket)
	c.KafkaBrokers = GetEnvOrFallback("KAFKA_BROKERS", c.KafkaBrokers)
	c.KafkaTopic = GetEnvOrFallback("KAFKA_TOPIC", c.KafkaTopic)
	c.ApplyMigrations = GetEnvOrFallback("APPLY_DB_MIGRATIONS", c.ApplyMigrations)

	tlsCertFile := GetEnvOrFallback("TLS_CERT_FILE", "")
	tlsKeyFile := GetEnvOrFallback("TLS_KEY_FILE", "")
	if tlsCertFile != "" || tlsKeyFile != "" {
		c.TLSConfig = &TLSConfig{
			CertFile: tlsCertFile,
			KeyFile:  tlsKeyFile,
		}
	}

	return nil
}

// UpdateFromCLIArgs updates settings from CLI arguments
func (c *ServerConfig) UpdateFromCLIArgs(flagSet *flag.FlagSet, args []string) error {
	flagSet.BoolVar(&c.ApplyMigrations, "apply-db-migrations", c.ApplyMigrations, "applies database migrations on server start")
	flagSet.BoolVar(&c.S3UseSSL, "s3-use-ssl", c.S3UseSSL, "turns on usage of SSL for S3 connections")

	flagSet.StringVar(&c.PsqlDSN, "d", c.PsqlDSN, "SQL database DSN string")
	flagSet.StringVar(&c.ListenAddr, "a", c.ListenAddr, "server listen address via HTTP")
	flagSet.StringVar(&c.S3ListenAddr, "g", c.S3ListenAddr, "S3 listen address")
	flagSet.StringVar(&c.S3AccessKey, "s3-access-key", c.S3AccessKey, "S3 access key")
	flagSet.StringVar(&c.S3SecretKey, "s3-secret-key", c.S3SecretKey, "S3 secret key")
	flagSet.StringVar(&c.S3Bucket, "s3-bucket", c.S3Bucket, "S3 bucket")
	flagSet.StringVar(&c.KafkaBrokers, "kafka-brokers", c.KafkaBrokers, "Kafka broker addresses (comma separated)")
	flagSet.StringVar(&c.KafkaTopic, "kafka-topic", c.KafkaTopic, "Kafka topic for avatar processing")

	// Using temporary variables to prevent nil pointer dereference if TLSConfig is nil
	var tlsCertFile string
	var tlsKeyFile string
	flagSet.StringVar(&tlsCertFile, "tls-cert", "", "path to TLS certificate file")
	flagSet.StringVar(&tlsKeyFile, "tls-key", "", "path to TLS private key file")

	var dummy string
	flagSet.StringVar(&dummy, "c", "", "path to configuration file")
	flagSet.StringVar(&dummy, "config", "", "path to the configurtion file")

	if err := flagSet.Parse(args); err != nil {
		return err
	}

	if tlsCertFile != "" || tlsKeyFile != "" {
		c.TLSConfig = &TLSConfig{
			CertFile: tlsCertFile,
			KeyFile:  tlsKeyFile,
		}
	} else {
		c.TLSConfig = nil
	}

	return nil
}

// UpdateFromFile checks the config file, and if it exists then tries to parse it
func (c *ServerConfig) UpdateFromFile(configFile string) error {
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

	var jsonCfg serverConfigJSON
	if err := json.Unmarshal(data, &jsonCfg); err != nil {
		return err
	}

	if jsonCfg.Address != "" {
		c.ListenAddr = jsonCfg.Address
	}

	if jsonCfg.DatabaseDSN != "" {
		c.PsqlDSN = jsonCfg.DatabaseDSN
	}

	if jsonCfg.S3Address != "" {
		c.S3ListenAddr = jsonCfg.S3Address
	}

	if jsonCfg.S3AccessKey != "" {
		c.S3AccessKey = jsonCfg.S3AccessKey
	}

	if jsonCfg.S3Secretkey != "" {
		c.S3SecretKey = jsonCfg.S3Secretkey
	}

	if jsonCfg.S3Bucket != "" {
		c.S3Bucket = jsonCfg.S3Bucket
	}

	if jsonCfg.S3UseSSL != c.S3UseSSL {
		c.S3UseSSL = jsonCfg.S3UseSSL
	}

	if jsonCfg.KafkaBrokers != "" {
		c.KafkaBrokers = jsonCfg.KafkaBrokers
	}

	if jsonCfg.KafkaTopic != "" {
		c.KafkaTopic = jsonCfg.KafkaTopic
	}

	if jsonCfg.TLS != nil {
		c.TLSConfig = &TLSConfig{
			CertFile: jsonCfg.TLS.CertFile,
			KeyFile:  jsonCfg.TLS.KeyFile,
		}
	} else {
		c.TLSConfig = nil
	}

	return nil
}

// Returns the instance of server configuration struct.
//
// Fills it's fields by using CLI arguments and environments.
// ENV or CLI argument or the default values
func NewServerConfig(flagSet *flag.FlagSet, args []string) (*ServerConfig, error) {
	cfg := GetDefaultServerConfig()
	configFile := GetEnvOrFallback("CONFIG", ParseConfigPath(args))

	if configFile != "" {
		if err := cfg.UpdateFromFile(configFile); err != nil {
			return nil, err
		}
	}

	if err := cfg.UpdateFromCLIArgs(flagSet, args); err != nil {
		return nil, err
	}

	if err := cfg.UpdateFromEnvironment(); err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}
