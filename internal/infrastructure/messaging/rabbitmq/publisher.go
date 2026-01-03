package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/felixgeelhaar/bolt"
	"github.com/felixgeelhaar/bridge/internal/domain/workflow"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Publisher publishes events to RabbitMQ.
type Publisher struct {
	conn     *Connection
	logger   *bolt.Logger
	exchange string
}

// PublisherConfig holds publisher configuration.
type PublisherConfig struct {
	Exchange     string
	ExchangeType string
	Durable      bool
	AutoDelete   bool
}

// DefaultPublisherConfig returns default publisher configuration.
func DefaultPublisherConfig() *PublisherConfig {
	return &PublisherConfig{
		Exchange:     "bridge.events",
		ExchangeType: "topic",
		Durable:      true,
		AutoDelete:   false,
	}
}

// NewPublisher creates a new RabbitMQ publisher.
func NewPublisher(conn *Connection, cfg *PublisherConfig, logger *bolt.Logger) (*Publisher, error) {
	ch := conn.Channel()

	// Declare exchange
	if err := ch.ExchangeDeclare(
		cfg.Exchange,
		cfg.ExchangeType,
		cfg.Durable,
		cfg.AutoDelete,
		false, // internal
		false, // no-wait
		nil,   // arguments
	); err != nil {
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	logger.Info().
		Str("exchange", cfg.Exchange).
		Str("type", cfg.ExchangeType).
		Msg("RabbitMQ exchange declared")

	return &Publisher{
		conn:     conn,
		logger:   logger,
		exchange: cfg.Exchange,
	}, nil
}

// Publish publishes an event to RabbitMQ.
func (p *Publisher) Publish(ctx context.Context, event workflow.Event) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	routingKey := event.EventType()

	msg := amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now(),
		MessageId:    event.EventID(),
		Type:         event.EventType(),
		Body:         body,
		Headers: amqp.Table{
			"aggregate_id": event.AggregateID(),
		},
	}

	ch := p.conn.Channel()
	if err := ch.PublishWithContext(
		ctx,
		p.exchange,
		routingKey,
		false, // mandatory
		false, // immediate
		msg,
	); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	p.logger.Debug().
		Str("event_id", event.EventID()).
		Str("event_type", event.EventType()).
		Str("routing_key", routingKey).
		Msg("Event published to RabbitMQ")

	return nil
}

// PublishJSON publishes a JSON message with a custom routing key.
func (p *Publisher) PublishJSON(ctx context.Context, routingKey string, data any) error {
	body, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	msg := amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now(),
		Body:         body,
	}

	ch := p.conn.Channel()
	if err := ch.PublishWithContext(
		ctx,
		p.exchange,
		routingKey,
		false,
		false,
		msg,
	); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	p.logger.Debug().
		Str("routing_key", routingKey).
		Msg("Message published to RabbitMQ")

	return nil
}

// Close is a no-op for publisher (connection manages lifecycle).
func (p *Publisher) Close() error {
	return nil
}
