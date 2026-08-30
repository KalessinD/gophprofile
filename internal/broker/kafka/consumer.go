package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/IBM/sarama"
	"github.com/KalessinD/gophprofile/internal/broker"
	"github.com/KalessinD/gophprofile/internal/logger"
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
		logger  logger.Logger
	}

	// consumerGroupHandler implements sarama.ConsumerGroupHandler interface.
	consumerGroupHandler struct {
		handler   Handler
		ready     chan bool
		readyOnce sync.Once
		logger    logger.Logger
	}
)

// NewConsumer initializes a new Kafka consumer group.
func NewConsumer(brokers string, topic string, groupID string, logger logger.Logger) (*Consumer, error) {
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

	c.wg.Go(func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				group := &consumerGroupHandler{
					handler: c.handler,
					ready:   c.ready,
					logger:  c.logger,
				}
				if err := c.client.Consume(ctx, []string{c.topic}, group); err != nil {
					c.logger.Errorf("Kafka consume error: %v", err)
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
	h.readyOnce.Do(func() {
		close(h.ready)
	})
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
				h.logger.Error("failed to unmarshal kafka message", "error", err)
				session.MarkMessage(msg, "")
				continue
			}

			if err := h.handler(session.Context(), &event); err != nil {
				h.logger.Error("failed to process avatar event", "error", err)
			}

			session.MarkMessage(msg, "")
		}
	}
}
