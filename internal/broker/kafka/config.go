package kafka

import (
	"time"

	"github.com/IBM/sarama"
	"github.com/KalessinD/gophprofile/internal/config"
)

// ConvertToSaramaConfig creates a new Sarama configuration with predefined settings
// for the synchronous producer.
func ConvertToSaramaConfig(_ *config.Kafka) *sarama.Config {
	saramaConfig := sarama.NewConfig()

	// let's consider to add these olptions into config.Kafka to be able to setup them
	saramaConfig.Producer.RequiredAcks = sarama.WaitForLocal
	saramaConfig.Producer.Compression = sarama.CompressionZSTD
	saramaConfig.Producer.Return.Successes = true
	saramaConfig.Producer.Timeout = 5 * time.Second
	saramaConfig.Producer.Retry.Max = 10
	saramaConfig.Metadata.RefreshFrequency = 1024

	return saramaConfig
}
