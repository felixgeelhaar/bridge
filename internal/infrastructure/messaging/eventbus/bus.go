package eventbus

import (
	"context"
	"sync"

	"github.com/felixgeelhaar/bridge/internal/domain/workflow"
)

// Handler is a function that handles an event.
type Handler func(ctx context.Context, event workflow.Event) error

// EventBus is an in-memory event bus for domain events.
type EventBus struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
	async    bool
}

// Option configures the event bus.
type Option func(*EventBus)

// WithAsync enables asynchronous event dispatch.
func WithAsync(async bool) Option {
	return func(b *EventBus) {
		b.async = async
	}
}

// New creates a new in-memory event bus.
func New(opts ...Option) *EventBus {
	bus := &EventBus{
		handlers: make(map[string][]Handler),
	}

	for _, opt := range opts {
		opt(bus)
	}

	return bus
}

// Subscribe registers a handler for a specific event type.
func (b *EventBus) Subscribe(eventType string, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

// SubscribeAll registers a handler for all event types.
func (b *EventBus) SubscribeAll(handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers["*"] = append(b.handlers["*"], handler)
}

// Publish publishes an event to all subscribed handlers.
func (b *EventBus) Publish(ctx context.Context, event workflow.Event) error {
	b.mu.RLock()
	handlers := make([]Handler, 0)

	// Get handlers for the specific event type
	if h, ok := b.handlers[event.EventType()]; ok {
		handlers = append(handlers, h...)
	}

	// Get handlers for all events
	if h, ok := b.handlers["*"]; ok {
		handlers = append(handlers, h...)
	}
	b.mu.RUnlock()

	if len(handlers) == 0 {
		return nil
	}

	if b.async {
		// Dispatch asynchronously
		for _, handler := range handlers {
			go func(h Handler) {
				_ = h(ctx, event) // Errors are silently ignored in async mode
			}(handler)
		}
		return nil
	}

	// Dispatch synchronously
	for _, handler := range handlers {
		if err := handler(ctx, event); err != nil {
			return err
		}
	}

	return nil
}

// PublishAsync publishes an event asynchronously, regardless of bus configuration.
func (b *EventBus) PublishAsync(ctx context.Context, event workflow.Event) {
	b.mu.RLock()
	handlers := make([]Handler, 0)

	if h, ok := b.handlers[event.EventType()]; ok {
		handlers = append(handlers, h...)
	}
	if h, ok := b.handlers["*"]; ok {
		handlers = append(handlers, h...)
	}
	b.mu.RUnlock()

	for _, handler := range handlers {
		go func(h Handler) {
			_ = h(ctx, event)
		}(handler)
	}
}

// Clear removes all handlers.
func (b *EventBus) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers = make(map[string][]Handler)
}

// Ensure EventBus implements workflow.EventPublisher.
var _ workflow.EventPublisher = (*EventBus)(nil)
