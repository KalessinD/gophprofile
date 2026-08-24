package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/IBM/sarama"
	"github.com/KalessinD/gophprofile/internal/broker"
	"go.uber.org/zap"
)

type (
	Handler func(ctx context.Context, event *broker.AvatarEvent) error

	// Consumer implements broker.EventConsumer using Apache Kafka Consumer Groups.
	Consumer struct {
		client  sarama.ConsumerGroup
		topic   string
		handler func(ctx context.Context, event *broker.AvatarEvent) error
		ready   chan bool
		wg      sync.WaitGroup
		logger  *zap.Logger
	}

	// consumerGroupHandler implements sarama.ConsumerGroupHandler interface.
	consumerGroupHandler struct {
		handler func(ctx context.Context, event *broker.AvatarEvent) error
		ready   chan bool
	}
)

// NewConsumer initializes a new Kafka consumer group.
func NewConsumer(brokers string, topic string, groupID string, logger *zap.Logger) (*Consumer, error) {
	config := sarama.NewConfig()
	config.Consumer.Group.Rebalance.GroupStrategies = append(config.Consumer.Group.Rebalance.GroupStrategies, sarama.NewBalanceStrategyRoundRobin())
	config.Version = sarama.DefaultVersion

	client, err := sarama.NewConsumerGroup([]string{brokers}, groupID, config)
	if err != nil {
		return nil, fmt.Errorf("creating kafka consumer group: %w", err)
	}

	return &Consumer{
		client: client,
		topic:  topic,
		ready:  make(chan bool),
		logger: logger,
	}, nil
}

// ConsumeAvatarEvents starts consuming messages from the Kafka topic in a background goroutine.
func (c *Consumer) ConsumeAvatarEvents(ctx context.Context, handler Handler) {
	c.handler = handler
	log := c.logger.Sugar()

	c.wg.Go(func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := c.client.Consume(ctx, []string{c.topic}, &consumerGroupHandler{handler: c.handler, ready: c.ready}); err != nil {
					log.Errorf("Kafka consume error: %v\n", err)
				}
				if ctx.Err() != nil {
					return
				}
			}
		}
	})
}

// Close gracefully shuts down the Kafka consumer and waits for the consumption loop to finish.
func (c *Consumer) Close() error {
	c.wg.Wait()
	if err := c.client.Close(); err != nil {
		return fmt.Errorf("closing kafka consumer: %w", err)
	}
	return nil
}

// Setup is called when a new session starts and is ready to consume messages.
func (h *consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	close(h.ready)
	return nil
}

// Cleanup is called when the session ends.
func (h *consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim processes messages from a single partition of the topic.
func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case <-session.Context().Done():
			return nil
		case msg, ok := <-claim.Messages():
			if !ok {
				return nil
			}

			var event broker.AvatarEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				// In a real app, log error here: log.Error("failed to unmarshal kafka message", zap.Error(err))
				session.MarkMessage(msg, "")
				continue
			}

			if err := h.handler(session.Context(), &event); err != nil {
				// In a real app, log error here: log.Error("failed to process avatar event", zap.Error(err))
				_ = err
			}

			// TODO: In the future, implement retry logic with a counter wrapper (e.g., {job: ..., counter: 1}).
			// If the counter exceeds the limit, log and discard (or route to DLQ).
			// For now, we always commit the offset (Variant B) and rely on the handler
			// to set the avatar status to "error" in the database if processing fails.
			session.MarkMessage(msg, "")
		}
	}
}
