package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"

	"golang-sms-broadcast/internal/domain"

	amqp "github.com/rabbitmq/amqp091-go"
)

const exchangeName = "sms"
const queueName = "sms.send"
const routingKey = "sms.send"

// Publisher implements ports.MessagePublisher using RabbitMQ.
type Publisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

// NewPublisher dials RabbitMQ, declares the exchange and queue, and binds them.
func NewPublisher(amqpURL string) (*Publisher, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, fmt.Errorf("dial rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("open channel: %w", err)
	}

	if err := declare(ch); err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}

	return &Publisher{conn: conn, channel: ch}, nil
}

// Publish serialises a domain.Message and sends it to the queue.
func (p *Publisher) Publish(ctx context.Context, msg domain.Message) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	return p.channel.PublishWithContext(
		ctx,
		exchangeName,
		routingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    msg.ID.String(),
			Body:         body,
		},
	)
}

// Close cleanly shuts down the channel and connection.
func (p *Publisher) Close() {
	p.channel.Close()
	p.conn.Close()
}

// declare idempotently sets up the exchange, queue, and binding.
func declare(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(exchangeName, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare exchange: %w", err)
	}

	if _, err := ch.QueueDeclare(queueName, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare queue: %w", err)
	}

	if err := ch.QueueBind(queueName, routingKey, exchangeName, false, nil); err != nil {
		return fmt.Errorf("bind queue: %w", err)
	}

	return nil
}
