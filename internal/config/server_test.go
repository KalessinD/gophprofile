package config_test

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/KalessinD/gophprofile/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDefaultServerConfig(t *testing.T) {
	cfg := config.GetDefaultServerConfig()

	require.NotNil(t, cfg)
	assert.Equal(t, config.DefaultListenAddr, cfg.ListenAddr)
	assert.NotNil(t, cfg.S3)
	assert.NotNil(t, cfg.Kafka)
}

func TestValidate_ValidAddress(t *testing.T) {
	cfg := config.GetDefaultServerConfig()
	err := cfg.Validate()
	assert.NoError(t, err)
}

func TestValidate_InvalidAddress(t *testing.T) {
	cfg := config.GetDefaultServerConfig()
	cfg.ListenAddr = "invalid-host"
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "address must be in format")
}

func TestValidate_InvalidS3Address(t *testing.T) {
	cfg := config.GetDefaultServerConfig()
	cfg.S3.ListenAddr = "minio:99999" // Port out of range
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "port must be between 1 and 65535")
}

func TestValidate_MissingTLSKey(t *testing.T) {
	cfg := config.GetDefaultServerConfig()
	cfg.TLSConfig = &config.TLSConfig{CertFile: "cert.pem"} // Missing KeyFile
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "both TLS certificate and key files must be provided")
}

func TestUpdateFromEnvironment(t *testing.T) {
	cfg := config.GetDefaultServerConfig()

	t.Setenv("ADDRESS", ":9090")
	t.Setenv("S3_ENDPOINT", "minio:9000")
	t.Setenv("S3_USE_SSL", "true")
	t.Setenv("KAFKA_BROKERS", "kafka:9092")

	err := cfg.UpdateFromEnvironment()
	require.NoError(t, err)

	assert.Equal(t, ":9090", cfg.ListenAddr)
	assert.Equal(t, "minio:9000", cfg.S3.ListenAddr)
	assert.True(t, cfg.S3.UseSSL)
	assert.Equal(t, "kafka:9092", cfg.Kafka.Brokers)
}

func TestUpdateFromCLIArgs(t *testing.T) {
	cfg := config.GetDefaultServerConfig()
	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	args := []string{"-a=:8081", "-s3-use-ssl", "-kafka-brokers=k1:9092,k2:9092"}
	err := cfg.UpdateFromCLIArgs(fs, args)
	require.NoError(t, err)

	assert.Equal(t, ":8081", cfg.ListenAddr)
	assert.True(t, cfg.S3.UseSSL)
	assert.Equal(t, "k1:9092,k2:9092", cfg.Kafka.Brokers)
}

func TestUpdateFromFile(t *testing.T) {
	cfg := config.GetDefaultServerConfig()

	jsonContent := `{
        "address": ":7070",
        "s3_address": "s3.local:9000",
        "kafka_brokers": "kafka.local:9092"
    }`

	tmpFile := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(jsonContent), 0o600))

	err := cfg.UpdateFromFile(tmpFile)
	require.NoError(t, err)

	assert.Equal(t, ":7070", cfg.ListenAddr)
	assert.Equal(t, "s3.local:9000", cfg.S3.ListenAddr)
	assert.Equal(t, "kafka.local:9092", cfg.Kafka.Brokers)
}

func TestNewServerConfig_PriorityChain(t *testing.T) {
	// File (Lowest priority)
	jsonContent := `{"address": ":1111", "s3_use_ssl": false}`
	tmpFile := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(jsonContent), 0o600))
	t.Setenv("CONFIG", tmpFile)

	// Environment (Medium priority)
	t.Setenv("ADDRESS", ":2222")
	t.Setenv("S3_USE_SSL", "true")

	// CLI (Highest priority)
	args := []string{"-a=:3333", "-s3-use-ssl=false"}

	cfg, err := config.NewServerConfig(flag.CommandLine, args)
	require.NoError(t, err)

	// CLI should override ENV and File
	assert.Equal(t, ":3333", cfg.ListenAddr)
	assert.False(t, cfg.S3.UseSSL)
}
