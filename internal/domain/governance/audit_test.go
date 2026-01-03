package governance

import (
	"context"
	"testing"
	"time"
)

func TestNewAuditEvent(t *testing.T) {
	event := NewAuditEvent(
		AuditEventWorkflowStarted,
		"user@example.com",
		"workflow",
		"wf-123",
		"start",
	)

	if event == nil {
		t.Fatal("NewAuditEvent returned nil")
	}

	if event.ID == "" {
		t.Error("Event ID should not be empty")
	}

	if event.Type != AuditEventWorkflowStarted {
		t.Errorf("Event type = %v, want %v", event.Type, AuditEventWorkflowStarted)
	}

	if event.Actor != "user@example.com" {
		t.Errorf("Actor = %v, want user@example.com", event.Actor)
	}

	if event.ResourceType != "workflow" {
		t.Errorf("ResourceType = %v, want workflow", event.ResourceType)
	}

	if event.ResourceID != "wf-123" {
		t.Errorf("ResourceID = %v, want wf-123", event.ResourceID)
	}

	if event.Action != "start" {
		t.Errorf("Action = %v, want start", event.Action)
	}

	if event.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestAuditEvent_WithDetails(t *testing.T) {
	event := NewAuditEvent(AuditEventWorkflowStarted, "user", "workflow", "123", "start").
		WithDetails("key1", "value1").
		WithDetails("key2", 42)

	if event.Details["key1"] != "value1" {
		t.Error("WithDetails failed for string value")
	}

	if event.Details["key2"] != 42 {
		t.Error("WithDetails failed for int value")
	}
}

func TestAuditEvent_WithTrace(t *testing.T) {
	event := NewAuditEvent(AuditEventWorkflowStarted, "user", "workflow", "123", "start").
		WithTrace("trace-123", "span-456")

	if event.TraceID != "trace-123" {
		t.Errorf("TraceID = %v, want trace-123", event.TraceID)
	}

	if event.SpanID != "span-456" {
		t.Errorf("SpanID = %v, want span-456", event.SpanID)
	}
}

func TestInMemoryAuditLogger_Log(t *testing.T) {
	logger := NewInMemoryAuditLogger()
	ctx := context.Background()

	event := NewAuditEvent(AuditEventWorkflowStarted, "user", "workflow", "123", "start")

	err := logger.Log(ctx, event)
	if err != nil {
		t.Fatalf("Log() error = %v", err)
	}

	// Query to verify
	events, err := logger.Query(ctx, AuditFilter{})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}

	if len(events) != 1 {
		t.Errorf("Query() returned %d events, want 1", len(events))
	}
}

func TestInMemoryAuditLogger_Query(t *testing.T) {
	logger := NewInMemoryAuditLogger()
	ctx := context.Background()

	// Add multiple events
	logger.Log(ctx, NewAuditEvent(AuditEventWorkflowStarted, "user1", "workflow", "wf-1", "start"))
	logger.Log(ctx, NewAuditEvent(AuditEventWorkflowCompleted, "system", "workflow", "wf-1", "complete"))
	logger.Log(ctx, NewAuditEvent(AuditEventWorkflowStarted, "user2", "workflow", "wf-2", "start"))
	logger.Log(ctx, NewAuditEvent(AuditEventPolicyViolation, "system", "workflow", "wf-2", "violation"))

	tests := []struct {
		name   string
		filter AuditFilter
		want   int
	}{
		{
			name:   "no filter",
			filter: AuditFilter{},
			want:   4,
		},
		{
			name:   "filter by type",
			filter: AuditFilter{Types: []AuditEventType{AuditEventWorkflowStarted}},
			want:   2,
		},
		{
			name:   "filter by actor",
			filter: AuditFilter{Actor: "user1"},
			want:   1,
		},
		{
			name:   "filter by resource ID",
			filter: AuditFilter{ResourceID: "wf-1"},
			want:   2,
		},
		{
			name:   "filter by resource type",
			filter: AuditFilter{ResourceType: "workflow"},
			want:   4,
		},
		{
			name:   "limit results",
			filter: AuditFilter{Limit: 2},
			want:   2,
		},
		{
			name:   "offset results",
			filter: AuditFilter{Offset: 2},
			want:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events, err := logger.Query(ctx, tt.filter)
			if err != nil {
				t.Fatalf("Query() error = %v", err)
			}
			if len(events) != tt.want {
				t.Errorf("Query() returned %d events, want %d", len(events), tt.want)
			}
		})
	}
}

func TestInMemoryAuditLogger_QueryTimeFilter(t *testing.T) {
	logger := NewInMemoryAuditLogger()
	ctx := context.Background()

	// Add event
	logger.Log(ctx, NewAuditEvent(AuditEventWorkflowStarted, "user", "workflow", "123", "start"))

	// Query with future start time should return 0
	futureTime := time.Now().Add(1 * time.Hour)
	events, _ := logger.Query(ctx, AuditFilter{StartTime: &futureTime})
	if len(events) != 0 {
		t.Error("Query with future start time should return 0 events")
	}

	// Query with past start time should return 1
	pastTime := time.Now().Add(-1 * time.Hour)
	events, _ = logger.Query(ctx, AuditFilter{StartTime: &pastTime})
	if len(events) != 1 {
		t.Error("Query with past start time should return 1 event")
	}
}

func TestAuditService_LogWorkflowStarted(t *testing.T) {
	logger := NewInMemoryAuditLogger()
	service := NewAuditService(logger)
	ctx := context.Background()

	err := service.LogWorkflowStarted(ctx, "wf-1", "run-1", "user@example.com")
	if err != nil {
		t.Fatalf("LogWorkflowStarted() error = %v", err)
	}

	events, _ := logger.Query(ctx, AuditFilter{Types: []AuditEventType{AuditEventWorkflowStarted}})
	if len(events) != 1 {
		t.Fatal("Expected 1 event")
	}

	if events[0].ResourceID != "run-1" {
		t.Errorf("ResourceID = %v, want run-1", events[0].ResourceID)
	}

	if events[0].Actor != "user@example.com" {
		t.Errorf("Actor = %v, want user@example.com", events[0].Actor)
	}
}

func TestAuditService_LogWorkflowCompleted(t *testing.T) {
	logger := NewInMemoryAuditLogger()
	service := NewAuditService(logger)
	ctx := context.Background()

	err := service.LogWorkflowCompleted(ctx, "wf-1", "run-1", 5*time.Second)
	if err != nil {
		t.Fatalf("LogWorkflowCompleted() error = %v", err)
	}

	events, _ := logger.Query(ctx, AuditFilter{Types: []AuditEventType{AuditEventWorkflowCompleted}})
	if len(events) != 1 {
		t.Fatal("Expected 1 event")
	}

	if events[0].Details["duration_ms"] != int64(5000) {
		t.Errorf("Duration = %v, want 5000", events[0].Details["duration_ms"])
	}
}

func TestAuditService_LogWorkflowFailed(t *testing.T) {
	logger := NewInMemoryAuditLogger()
	service := NewAuditService(logger)
	ctx := context.Background()

	err := service.LogWorkflowFailed(ctx, "wf-1", "run-1", "something went wrong")
	if err != nil {
		t.Fatalf("LogWorkflowFailed() error = %v", err)
	}

	events, _ := logger.Query(ctx, AuditFilter{Types: []AuditEventType{AuditEventWorkflowFailed}})
	if len(events) != 1 {
		t.Fatal("Expected 1 event")
	}

	if events[0].Details["error"] != "something went wrong" {
		t.Error("Error message not recorded correctly")
	}
}

func TestAuditService_LogPolicyEvaluated(t *testing.T) {
	logger := NewInMemoryAuditLogger()
	service := NewAuditService(logger)
	ctx := context.Background()

	err := service.LogPolicyEvaluated(ctx, "run-1", "security-policy", true)
	if err != nil {
		t.Fatalf("LogPolicyEvaluated() error = %v", err)
	}

	events, _ := logger.Query(ctx, AuditFilter{Types: []AuditEventType{AuditEventPolicyEvaluated}})
	if len(events) != 1 {
		t.Fatal("Expected 1 event")
	}

	if events[0].Details["allowed"] != true {
		t.Error("Allowed flag not recorded correctly")
	}
}

func TestAuditService_LogApprovalGranted(t *testing.T) {
	logger := NewInMemoryAuditLogger()
	service := NewAuditService(logger)
	ctx := context.Background()

	err := service.LogApprovalGranted(ctx, "approval-1", "run-1", "admin@example.com")
	if err != nil {
		t.Fatalf("LogApprovalGranted() error = %v", err)
	}

	events, _ := logger.Query(ctx, AuditFilter{Types: []AuditEventType{AuditEventApprovalGranted}})
	if len(events) != 1 {
		t.Fatal("Expected 1 event")
	}

	if events[0].Actor != "admin@example.com" {
		t.Errorf("Actor = %v, want admin@example.com", events[0].Actor)
	}
}

func TestAuditEventTypes(t *testing.T) {
	// Verify all event type constants have correct values
	types := map[AuditEventType]string{
		AuditEventWorkflowStarted:   "workflow.started",
		AuditEventWorkflowCompleted: "workflow.completed",
		AuditEventWorkflowFailed:    "workflow.failed",
		AuditEventStepExecuted:      "step.executed",
		AuditEventPolicyEvaluated:   "policy.evaluated",
		AuditEventPolicyViolation:   "policy.violation",
		AuditEventApprovalRequested: "approval.requested",
		AuditEventApprovalGranted:   "approval.granted",
		AuditEventApprovalRejected:  "approval.rejected",
		AuditEventToolInvoked:       "tool.invoked",
		AuditEventAgentCalled:       "agent.called",
	}

	for eventType, expected := range types {
		if string(eventType) != expected {
			t.Errorf("Event type %v = %v, want %v", eventType, string(eventType), expected)
		}
	}
}
