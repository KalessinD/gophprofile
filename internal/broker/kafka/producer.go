package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/IBM/sarama"
	"github.com/KalessinD/gophprofile/internal/broker"
)

type (
	// Producer implements broker.EventPublisher using Apache Kafka.
	Producer struct {
		producer sarama.AsyncProducer
		topic    string
	}
)

// NewProducer initializes a new Kafka producer.
func NewProducer(brokers string, topic string) (*Producer, error) {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForLocal
	config.Producer.Compression = sarama.CompressionZSTD
	config.Producer.Return.Successes = true
	config.Producer.Timeout = 5 * time.Second
	config.Producer.Flush.Frequency = time.Millisecond * 50
	config.Producer.Flush.MaxMessages = 100
	config.Producer.Flush.Messages = 50
	config.Producer.Retry.Max = 10
	config.Metadata.RefreshFrequency = 1024

	producer, err := sarama.NewAsyncProducer([]string{brokers}, config)
	if err != nil {
		return nil, fmt.Errorf("creating kafka producer: %w", err)
	}

	return &Producer{
		producer: producer,
		topic:    topic,
	}, nil
}

// PublishAvatarCreatedEvent serializes the event and sends it to the Kafka topic.
func (p *Producer) PublishAvatarCreatedEvent(ctx context.Context, event *broker.AvatarEvent) error {
	eventBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshaling avatar event: %w", err)
	}

	message := &sarama.ProducerMessage{
		Topic: p.topic,
		Value: sarama.ByteEncoder(eventBytes),
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case p.producer.Input() <- message:
		return nil
	case err := <-p.producer.Errors():
		return fmt.Errorf("kafka producer message: %w", err)
	}
}

// Close gracefully shuts down the Kafka producer.
func (p *Producer) Close() error {
	if err := p.producer.Close(); err != nil {
		return fmt.Errorf("closing kafka producer: %w", err)
	}
	return nil
}
