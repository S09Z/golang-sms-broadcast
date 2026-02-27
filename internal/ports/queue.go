package ports

import (
	"context"

	"golang-sms-broadcast/internal/domain"
)

// MessagePublisher publishes messages to the message queue.
type MessagePublisher interface {
	// Publish sends a single domain.Message to the queue.
	Publish(ctx context.Context, msg domain.Message) error
}

// MessageConsumer consumes messages from the message queue.
type MessageConsumer interface {
	// Consume starts delivery of messages; each is passed to the handler.
	// Blocks until ctx is cancelled or a fatal error occurs.
	Consume(ctx context.Context, handler func(ctx context.Context, msg domain.Message) error) error
}
