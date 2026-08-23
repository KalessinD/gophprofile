package broker

import (
	"context"
)

type (
	// AvatarEvent represents the structure of the message sent to Kafka for image processing.
	AvatarEvent struct {
		AvatarID string `json:"avatar_id"`
		UserID   string `json:"user_id"`
		S3Key    string `json:"s3_key"`
	}

	// EventPublisher defines the contract for publishing events to a message broker.
	EventPublisher interface {
		PublishAvatarCreatedEvent(ctx context.Context, event *AvatarEvent) error
		Close() error
	}

	// EventConsumer defines the contract for consuming events from a message broker.
	EventConsumer interface {
		ConsumeAvatarEvents(ctx context.Context, handler func(ctx context.Context, event *AvatarEvent) error)
		Close() error
	}
)
