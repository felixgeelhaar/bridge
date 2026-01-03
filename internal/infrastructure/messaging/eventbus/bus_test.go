package eventbus

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/felixgeelhaar/bridge/internal/domain/workflow"
	"github.com/felixgeelhaar/bridge/pkg/types"
)

// mockEvent implements workflow.Event for testing.
type mockEvent struct {
	id        string
	eventType string
	timestamp time.Time
	aggregate string
}

func (e *mockEvent) EventID() string       { return e.id }
func (e *mockEvent) EventType() string     { return e.eventType }
func (e *mockEvent) OccurredAt() time.Time { return e.timestamp }
func (e *mockEvent) AggregateID() string   { return e.aggregate }

func newMockEvent(eventType string) *mockEvent {
	return &mockEvent{
		id:        types.NewRunID().String(),
		eventType: eventType,
		timestamp: time.Now(),
		aggregate: "test-aggregate",
	}
}

func TestNew(t *testing.T) {
	bus := New()
	if bus == nil {
		t.Fatal("New() returned nil")
	}
	if bus.async {
		t.Error("Default bus should not be async")
	}
}

func TestNew_WithAsync(t *testing.T) {
	bus := New(WithAsync(true))
	if !bus.async {
		t.Error("Bus should be async when WithAsync(true) is used")
	}
}

func TestEventBus_Subscribe(t *testing.T) {
	bus := New()

	bus.Subscribe("test.event", func(ctx context.Context, event workflow.Event) error {
		return nil
	})

	if len(bus.handlers["test.event"]) != 1 {
		t.Error("Handler should be registered")
	}
}

func TestEventBus_SubscribeAll(t *testing.T) {
	bus := New()

	bus.SubscribeAll(func(ctx context.Context, event workflow.Event) error {
		return nil
	})

	if len(bus.handlers["*"]) != 1 {
		t.Error("SubscribeAll handler should be registered under '*'")
	}
}

func TestEventBus_Publish_Synchronous(t *testing.T) {
	bus := New()
	ctx := context.Background()

	var receivedEvents []workflow.Event
	bus.Subscribe("test.event", func(ctx context.Context, event workflow.Event) error {
		receivedEvents = append(receivedEvents, event)
		return nil
	})

	event := newMockEvent("test.event")
	err := bus.Publish(ctx, event)
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	if len(receivedEvents) != 1 {
		t.Error("Handler should receive the event")
	}

	if receivedEvents[0].EventType() != "test.event" {
		t.Error("Received event type should match")
	}
}

func TestEventBus_Publish_MultipleHandlers(t *testing.T) {
	bus := New()
	ctx := context.Background()

	var count int32
	handler := func(ctx context.Context, event workflow.Event) error {
		atomic.AddInt32(&count, 1)
		return nil
	}

	bus.Subscribe("test.event", handler)
	bus.Subscribe("test.event", handler)
	bus.Subscribe("test.event", handler)

	err := bus.Publish(ctx, newMockEvent("test.event"))
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	if atomic.LoadInt32(&count) != 3 {
		t.Errorf("Expected 3 handler calls, got %d", count)
	}
}

func TestEventBus_Publish_SubscribeAll(t *testing.T) {
	bus := New()
	ctx := context.Background()

	var receivedTypes []string
	bus.SubscribeAll(func(ctx context.Context, event workflow.Event) error {
		receivedTypes = append(receivedTypes, event.EventType())
		return nil
	})

	bus.Publish(ctx, newMockEvent("type1"))
	bus.Publish(ctx, newMockEvent("type2"))
	bus.Publish(ctx, newMockEvent("type3"))

	if len(receivedTypes) != 3 {
		t.Errorf("Expected 3 events, got %d", len(receivedTypes))
	}
}

func TestEventBus_Publish_BothSpecificAndAll(t *testing.T) {
	bus := New()
	ctx := context.Background()

	var specificCount, allCount int32

	bus.Subscribe("test.event", func(ctx context.Context, event workflow.Event) error {
		atomic.AddInt32(&specificCount, 1)
		return nil
	})

	bus.SubscribeAll(func(ctx context.Context, event workflow.Event) error {
		atomic.AddInt32(&allCount, 1)
		return nil
	})

	bus.Publish(ctx, newMockEvent("test.event"))

	if atomic.LoadInt32(&specificCount) != 1 {
		t.Error("Specific handler should be called once")
	}
	if atomic.LoadInt32(&allCount) != 1 {
		t.Error("SubscribeAll handler should be called once")
	}
}

func TestEventBus_Publish_NoHandlers(t *testing.T) {
	bus := New()
	ctx := context.Background()

	err := bus.Publish(ctx, newMockEvent("unhandled.event"))
	if err != nil {
		t.Error("Publishing to no handlers should not error")
	}
}

func TestEventBus_Publish_HandlerError(t *testing.T) {
	bus := New()
	ctx := context.Background()

	expectedErr := errors.New("handler error")
	bus.Subscribe("test.event", func(ctx context.Context, event workflow.Event) error {
		return expectedErr
	})

	err := bus.Publish(ctx, newMockEvent("test.event"))
	if err != expectedErr {
		t.Errorf("Publish() should return handler error, got %v", err)
	}
}

func TestEventBus_Publish_HandlerError_StopsExecution(t *testing.T) {
	bus := New()
	ctx := context.Background()

	var secondCalled bool
	bus.Subscribe("test.event", func(ctx context.Context, event workflow.Event) error {
		return errors.New("first handler error")
	})
	bus.Subscribe("test.event", func(ctx context.Context, event workflow.Event) error {
		secondCalled = true
		return nil
	})

	bus.Publish(ctx, newMockEvent("test.event"))

	if secondCalled {
		t.Error("Second handler should not be called when first errors")
	}
}

func TestEventBus_Publish_Async(t *testing.T) {
	bus := New(WithAsync(true))
	ctx := context.Background()

	var wg sync.WaitGroup
	wg.Add(1)

	var received bool
	bus.Subscribe("test.event", func(ctx context.Context, event workflow.Event) error {
		received = true
		wg.Done()
		return nil
	})

	err := bus.Publish(ctx, newMockEvent("test.event"))
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	wg.Wait()
	if !received {
		t.Error("Handler should receive the event asynchronously")
	}
}

func TestEventBus_PublishAsync(t *testing.T) {
	bus := New() // Not async by default
	ctx := context.Background()

	var wg sync.WaitGroup
	wg.Add(1)

	var received bool
	bus.Subscribe("test.event", func(ctx context.Context, event workflow.Event) error {
		received = true
		wg.Done()
		return nil
	})

	bus.PublishAsync(ctx, newMockEvent("test.event"))

	wg.Wait()
	if !received {
		t.Error("PublishAsync should dispatch asynchronously regardless of bus config")
	}
}

func TestEventBus_Clear(t *testing.T) {
	bus := New()

	bus.Subscribe("test1", func(ctx context.Context, event workflow.Event) error { return nil })
	bus.Subscribe("test2", func(ctx context.Context, event workflow.Event) error { return nil })
	bus.SubscribeAll(func(ctx context.Context, event workflow.Event) error { return nil })

	if len(bus.handlers) != 3 {
		t.Error("Should have 3 handler groups before clear")
	}

	bus.Clear()

	if len(bus.handlers) != 0 {
		t.Error("Should have no handlers after clear")
	}
}

func TestEventBus_ConcurrentAccess(t *testing.T) {
	bus := New()
	ctx := context.Background()

	var wg sync.WaitGroup

	// Concurrent subscriptions
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			bus.Subscribe("test.event", func(ctx context.Context, event workflow.Event) error {
				return nil
			})
		}(i)
	}

	// Concurrent publishing
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.Publish(ctx, newMockEvent("test.event"))
		}()
	}

	wg.Wait()
	// If we get here without race conditions, test passes
}

func TestEventBus_ImplementsEventPublisher(t *testing.T) {
	var _ workflow.EventPublisher = (*EventBus)(nil)
}
