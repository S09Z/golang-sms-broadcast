package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"golang-sms-broadcast/internal/domain"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Consumer implements ports.MessageConsumer using RabbitMQ.
type Consumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	log     *slog.Logger
}

// NewConsumer dials RabbitMQ, declares topology, and returns a Consumer.
func NewConsumer(amqpURL string, log *slog.Logger) (*Consumer, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, fmt.Errorf("dial rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("open channel: %w", err)
	}

	// One message at a time per consumer to ensure ordered processing.
	if err := ch.Qos(1, 0, false); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("set qos: %w", err)
	}

	if err := declare(ch); err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}

	return &Consumer{conn: conn, channel: ch, log: log}, nil
}

// Consume registers a consumer on the queue and calls handler for each delivery.
// It acknowledges the message only if the handler returns nil.
// It blocks until ctx is cancelled.
func (c *Consumer) Consume(ctx context.Context, handler func(ctx context.Context, msg domain.Message) error) error {
	deliveries, err := c.channel.Consume(
		queueName,
		"",    // auto-generated consumer tag
		false, // manual ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("consume: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case d, ok := <-deliveries:
			if !ok {
				return fmt.Errorf("deliveries channel closed")
			}

			var msg domain.Message
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				c.log.Error("unmarshal message", "err", err)
				d.Nack(false, false) // dead-letter; don't requeue malformed payloads
				continue
			}

			if err := handler(ctx, msg); err != nil {
				c.log.Error("handler error", "msg_id", msg.ID, "err", err)
				d.Nack(false, true) // requeue for retry
				continue
			}

			d.Ack(false)
		}
	}
}

// Close cleanly shuts down the channel and connection.
func (c *Consumer) Close() {
	c.channel.Close()
	c.conn.Close()
}
