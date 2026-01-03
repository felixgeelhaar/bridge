package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/felixgeelhaar/bolt"
	amqp "github.com/rabbitmq/amqp091-go"
)

// MessageHandler handles a received message.
type MessageHandler func(ctx context.Context, msg *Message) error

// Message represents a received message.
type Message struct {
	ID          string
	Type        string
	Body        []byte
	Headers     map[string]any
	RoutingKey  string
	Exchange    string
	Redelivered bool
	delivery    amqp.Delivery
}

// Unmarshal unmarshals the message body into the given value.
func (m *Message) Unmarshal(v any) error {
	return json.Unmarshal(m.Body, v)
}

// Ack acknowledges the message.
func (m *Message) Ack() error {
	return m.delivery.Ack(false)
}

// Nack negatively acknowledges the message.
func (m *Message) Nack(requeue bool) error {
	return m.delivery.Nack(false, requeue)
}

// Reject rejects the message.
func (m *Message) Reject(requeue bool) error {
	return m.delivery.Reject(requeue)
}

// Subscriber subscribes to RabbitMQ queues.
type Subscriber struct {
	conn      *Connection
	logger    *bolt.Logger
	exchange  string
	handlers  map[string]MessageHandler
	queues    map[string]string
	mu        sync.RWMutex
	consumers []string
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
}

// SubscriberConfig holds subscriber configuration.
type SubscriberConfig struct {
	Exchange    string
	QueuePrefix string
	Durable     bool
	AutoDelete  bool
	Exclusive   bool
}

// DefaultSubscriberConfig returns default subscriber configuration.
func DefaultSubscriberConfig() *SubscriberConfig {
	return &SubscriberConfig{
		Exchange:    "bridge.events",
		QueuePrefix: "bridge",
		Durable:     true,
		AutoDelete:  false,
		Exclusive:   false,
	}
}

// NewSubscriber creates a new RabbitMQ subscriber.
func NewSubscriber(conn *Connection, cfg *SubscriberConfig, logger *bolt.Logger) *Subscriber {
	ctx, cancel := context.WithCancel(context.Background())

	return &Subscriber{
		conn:     conn,
		logger:   logger,
		exchange: cfg.Exchange,
		handlers: make(map[string]MessageHandler),
		queues:   make(map[string]string),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Subscribe subscribes to events matching the routing key pattern.
func (s *Subscriber) Subscribe(routingKey string, queueName string, handler MessageHandler) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := s.conn.Channel()

	// Declare queue
	queue, err := ch.QueueDeclare(
		queueName,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Bind queue to exchange
	if err := ch.QueueBind(
		queue.Name,
		routingKey,
		s.exchange,
		false, // no-wait
		nil,   // arguments
	); err != nil {
		return fmt.Errorf("failed to bind queue: %w", err)
	}

	s.handlers[queueName] = handler
	s.queues[routingKey] = queueName

	s.logger.Info().
		Str("queue", queueName).
		Str("routing_key", routingKey).
		Str("exchange", s.exchange).
		Msg("Subscribed to RabbitMQ queue")

	return nil
}

// Start starts consuming messages from all subscribed queues.
func (s *Subscriber) Start() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ch := s.conn.Channel()

	for queueName, handler := range s.handlers {
		consumerTag := fmt.Sprintf("%s-consumer", queueName)

		deliveries, err := ch.Consume(
			queueName,
			consumerTag,
			false, // auto-ack
			false, // exclusive
			false, // no-local
			false, // no-wait
			nil,   // arguments
		)
		if err != nil {
			return fmt.Errorf("failed to start consumer for %s: %w", queueName, err)
		}

		s.consumers = append(s.consumers, consumerTag)
		s.wg.Add(1)

		go s.consume(queueName, handler, deliveries)

		s.logger.Info().
			Str("queue", queueName).
			Str("consumer", consumerTag).
			Msg("Started consuming from queue")
	}

	return nil
}

func (s *Subscriber) consume(queueName string, handler MessageHandler, deliveries <-chan amqp.Delivery) {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return
		case delivery, ok := <-deliveries:
			if !ok {
				s.logger.Warn().
					Str("queue", queueName).
					Msg("Delivery channel closed")
				return
			}

			msg := &Message{
				ID:          delivery.MessageId,
				Type:        delivery.Type,
				Body:        delivery.Body,
				Headers:     delivery.Headers,
				RoutingKey:  delivery.RoutingKey,
				Exchange:    delivery.Exchange,
				Redelivered: delivery.Redelivered,
				delivery:    delivery,
			}

			if err := handler(s.ctx, msg); err != nil {
				s.logger.Error().
					Err(err).
					Str("queue", queueName).
					Str("message_id", msg.ID).
					Msg("Failed to handle message")

				// Nack and requeue if not redelivered, otherwise dead-letter
				if err := msg.Nack(!msg.Redelivered); err != nil {
					s.logger.Error().Err(err).Msg("Failed to nack message")
				}
			} else {
				if err := msg.Ack(); err != nil {
					s.logger.Error().Err(err).Msg("Failed to ack message")
				}
			}
		}
	}
}

// Stop stops all consumers.
func (s *Subscriber) Stop() error {
	s.cancel()

	ch := s.conn.Channel()
	for _, consumerTag := range s.consumers {
		if err := ch.Cancel(consumerTag, false); err != nil {
			s.logger.Error().
				Err(err).
				Str("consumer", consumerTag).
				Msg("Failed to cancel consumer")
		}
	}

	s.wg.Wait()

	s.logger.Info().Msg("All RabbitMQ consumers stopped")
	return nil
}
