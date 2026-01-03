package rabbitmq

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/felixgeelhaar/bolt"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Config holds RabbitMQ connection configuration.
type Config struct {
	URL            string
	ReconnectDelay time.Duration
	MaxReconnects  int
	PrefetchCount  int
	PrefetchGlobal bool
}

// DefaultConfig returns a default configuration.
func DefaultConfig() *Config {
	return &Config{
		URL:            "amqp://guest:guest@localhost:5672/",
		ReconnectDelay: 5 * time.Second,
		MaxReconnects:  10,
		PrefetchCount:  10,
		PrefetchGlobal: false,
	}
}

// Connection manages a RabbitMQ connection with automatic reconnection.
type Connection struct {
	config     *Config
	logger     *bolt.Logger
	conn       *amqp.Connection
	channel    *amqp.Channel
	mu         sync.RWMutex
	closed     bool
	closeChan  chan struct{}
	notifyConn chan *amqp.Error
	notifyChan chan *amqp.Error
}

// NewConnection creates a new RabbitMQ connection.
func NewConnection(cfg *Config, logger *bolt.Logger) (*Connection, error) {
	c := &Connection{
		config:    cfg,
		logger:    logger,
		closeChan: make(chan struct{}),
	}

	if err := c.connect(); err != nil {
		return nil, err
	}

	go c.handleReconnect()

	return c, nil
}

func (c *Connection) connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	conn, err := amqp.Dial(c.config.URL)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("failed to open channel: %w", err)
	}

	if err := ch.Qos(c.config.PrefetchCount, 0, c.config.PrefetchGlobal); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	c.conn = conn
	c.channel = ch
	c.notifyConn = make(chan *amqp.Error, 1)
	c.notifyChan = make(chan *amqp.Error, 1)
	c.conn.NotifyClose(c.notifyConn)
	c.channel.NotifyClose(c.notifyChan)

	c.logger.Info().
		Str("url", c.config.URL).
		Msg("Connected to RabbitMQ")

	return nil
}

func (c *Connection) handleReconnect() {
	for {
		select {
		case <-c.closeChan:
			return
		case err := <-c.notifyConn:
			if err != nil {
				c.logger.Warn().Err(err).Msg("RabbitMQ connection closed")
				c.reconnect()
			}
		case err := <-c.notifyChan:
			if err != nil {
				c.logger.Warn().Err(err).Msg("RabbitMQ channel closed")
				c.reconnect()
			}
		}
	}
}

func (c *Connection) reconnect() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()

	for i := 0; i < c.config.MaxReconnects; i++ {
		c.logger.Info().
			Int("attempt", i+1).
			Int("max_attempts", c.config.MaxReconnects).
			Msg("Attempting to reconnect to RabbitMQ")

		if err := c.connect(); err != nil {
			c.logger.Error().Err(err).Msg("Failed to reconnect")
			time.Sleep(c.config.ReconnectDelay)
			continue
		}

		c.logger.Info().Msg("Successfully reconnected to RabbitMQ")
		return
	}

	c.logger.Error().Msg("Max reconnection attempts reached")
}

// Channel returns the current channel.
func (c *Connection) Channel() *amqp.Channel {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.channel
}

// Connection returns the underlying connection.
func (c *Connection) Connection() *amqp.Connection {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn
}

// Close closes the connection.
func (c *Connection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	close(c.closeChan)

	var errs []error

	if c.channel != nil {
		if err := c.channel.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing connection: %v", errs)
	}

	c.logger.Info().Msg("RabbitMQ connection closed")
	return nil
}

// HealthCheck returns true if the connection is healthy.
func (c *Connection) HealthCheck(ctx context.Context) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.conn == nil || c.conn.IsClosed() {
		return false
	}

	if c.channel == nil || c.channel.IsClosed() {
		return false
	}

	return true
}
