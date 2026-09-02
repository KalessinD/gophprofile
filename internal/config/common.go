package config

import (
	"flag"
	"strings"
	"time"
)

const (
	DefaultS3ListenAddr string = ":9000"
	DefaultS3UseSSL     bool   = false
	DefaultS3AccessKey  string = ""
	DefaultS3Secretkey  string = ""
	DefaultS3Bucket     string = "gophprofile"

	DefaultKafkaBrokers string = ""
	DefaultKafkaTopic   string = "avatar-processing"
	DefaultKafkaGroupID string = "gophprofile-worker-group"

	DefaultLoggerType string = "slog" // zap is possible too
	DefaultPsqlDSN    string = ""

	DefaultOTELExporterOTLPEndpoint string = "jaeger:4317"

	DefaultGracefullShutdownTimeout time.Duration = 5 * time.Second
	DefaultMetricReadPeriod         time.Duration = 2 * time.Second
)

var validLogTypes = map[string]struct{}{
	"slog": {},
	"zap":  {},
}

type (
	LoggerType string

	S3 struct {
		UseSSL     bool
		AccessKey  string
		SecretKey  string
		Bucket     string
		ListenAddr string
	}

	Kafka struct {
		Brokers string
		Topic   string
		GroupID string
	}

	Otel struct {
		ExporterOTLPEndpoint string
		MetricReadPeriod     time.Duration
		ShutdownTimeout      time.Duration
	}

	// Configurator is an interface to be implemented by server/worker configuration object
	Configurator interface {
		UpdateFromEnvironment() error
		UpdateFromCLIArgs(flagSet *flag.FlagSet, args []string) error
		UpdateFromFile(configFile string) error
		Validate() error
	}
)

func getDefaultOtel() *Otel {
	return &Otel{
		ExporterOTLPEndpoint: DefaultOTELExporterOTLPEndpoint,
		ShutdownTimeout:      DefaultGracefullShutdownTimeout,
		MetricReadPeriod:     DefaultMetricReadPeriod,
	}
}

func getDefaultS3() *S3 {
	return &S3{
		ListenAddr: DefaultS3ListenAddr,
		UseSSL:     DefaultS3UseSSL,
		AccessKey:  DefaultS3AccessKey,
		SecretKey:  DefaultS3Secretkey,
		Bucket:     DefaultS3Bucket,
	}
}

func getDefaultKafka() *Kafka {
	return &Kafka{
		Brokers: DefaultKafkaBrokers,
		Topic:   DefaultKafkaTopic,
		GroupID: DefaultKafkaGroupID,
	}
}

// ParseConfigPath extracts config file path from args without using flag package
func ParseConfigPath(args []string) string {
	for i, arg := range args {
		switch {
		case arg == "-c", arg == "-config":
			if i+1 < len(args) {
				return args[i+1]
			}
		case strings.HasPrefix(arg, "-c="):
			return strings.TrimPrefix(arg, "-c=")
		case strings.HasPrefix(arg, "-config="):
			return strings.TrimPrefix(arg, "-config=")
		}
	}
	return ""
}
