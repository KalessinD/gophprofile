package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/IBM/sarama"
	"github.com/KalessinD/gophprofile/internal/broker"
)

type (
	// Producer implements broker.EventPublisher using Apache Kafka.
	Producer struct {
		producer sarama.SyncProducer
		topic    string
	}
)

// NewProducer initializes a new Kafka synchronous producer.
func NewProducer(brokers string, topic string, saramaCfg *sarama.Config) (*Producer, error) {
	producer, err := sarama.NewSyncProducer([]string{brokers}, saramaCfg)
	if err != nil {
		return nil, fmt.Errorf("creating kafka producer: %w", err)
	}

	return &Producer{
		producer: producer,
		topic:    topic,
	}, nil
}

// PublishAvatarCreatedEvent serializes the event and sends it to the Kafka topic synchronously.
func (p *Producer) PublishAvatarCreatedEvent(_ context.Context, avatarID string, userID string, s3Key string) error {
	event := &broker.AvatarEvent{
		AvatarID: avatarID,
		UserID:   userID,
		S3Key:    s3Key,
	}

	eventBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshaling avatar event: %w", err)
	}

	message := &sarama.ProducerMessage{
		Topic: p.topic,
		Value: sarama.ByteEncoder(eventBytes),
	}

	_, _, err = p.producer.SendMessage(message)
	if err != nil {
		return fmt.Errorf("sending message to kafka: %w", err)
	}

	return nil
}

// Close gracefully shuts down the Kafka producer.
func (p *Producer) Close() error {
	if err := p.producer.Close(); err != nil {
		return fmt.Errorf("closing kafka producer: %w", err)
	}
	return nil
}
