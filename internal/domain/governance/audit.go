package governance

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// AuditEventType represents the type of audit event.
type AuditEventType string

const (
	AuditEventWorkflowStarted   AuditEventType = "workflow.started"
	AuditEventWorkflowCompleted AuditEventType = "workflow.completed"
	AuditEventWorkflowFailed    AuditEventType = "workflow.failed"
	AuditEventStepExecuted      AuditEventType = "step.executed"
	AuditEventPolicyEvaluated   AuditEventType = "policy.evaluated"
	AuditEventPolicyViolation   AuditEventType = "policy.violation"
	AuditEventApprovalRequested AuditEventType = "approval.requested"
	AuditEventApprovalGranted   AuditEventType = "approval.granted"
	AuditEventApprovalRejected  AuditEventType = "approval.rejected"
	AuditEventToolInvoked       AuditEventType = "tool.invoked"
	AuditEventAgentCalled       AuditEventType = "agent.called"
)

// AuditEvent represents an auditable event in the system.
type AuditEvent struct {
	ID           string         `json:"id"`
	Type         AuditEventType `json:"type"`
	Actor        string         `json:"actor"`
	ResourceType string         `json:"resource_type"`
	ResourceID   string         `json:"resource_id"`
	Action       string         `json:"action"`
	Details      map[string]any `json:"details,omitempty"`
	TraceID      string         `json:"trace_id,omitempty"`
	SpanID       string         `json:"span_id,omitempty"`
	Timestamp    time.Time      `json:"timestamp"`
}

// NewAuditEvent creates a new audit event.
func NewAuditEvent(eventType AuditEventType, actor, resourceType, resourceID, action string) *AuditEvent {
	return &AuditEvent{
		ID:           uuid.New().String(),
		Type:         eventType,
		Actor:        actor,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Action:       action,
		Details:      make(map[string]any),
		Timestamp:    time.Now(),
	}
}

// WithDetails adds details to the audit event.
func (e *AuditEvent) WithDetails(key string, value any) *AuditEvent {
	e.Details[key] = value
	return e
}

// WithTrace adds trace context to the audit event.
func (e *AuditEvent) WithTrace(traceID, spanID string) *AuditEvent {
	e.TraceID = traceID
	e.SpanID = spanID
	return e
}

// AuditLogger logs audit events.
type AuditLogger interface {
	Log(ctx context.Context, event *AuditEvent) error
	Query(ctx context.Context, filter AuditFilter) ([]*AuditEvent, error)
}

// AuditFilter filters audit events for queries.
type AuditFilter struct {
	Types        []AuditEventType
	Actor        string
	ResourceType string
	ResourceID   string
	StartTime    *time.Time
	EndTime      *time.Time
	Limit        int
	Offset       int
}

// InMemoryAuditLogger is an in-memory implementation of AuditLogger.
type InMemoryAuditLogger struct {
	events []*AuditEvent
}

// NewInMemoryAuditLogger creates a new in-memory audit logger.
func NewInMemoryAuditLogger() *InMemoryAuditLogger {
	return &InMemoryAuditLogger{
		events: make([]*AuditEvent, 0),
	}
}

// Log logs an audit event.
func (l *InMemoryAuditLogger) Log(ctx context.Context, event *AuditEvent) error {
	l.events = append(l.events, event)
	return nil
}

// Query queries audit events.
func (l *InMemoryAuditLogger) Query(ctx context.Context, filter AuditFilter) ([]*AuditEvent, error) {
	result := make([]*AuditEvent, 0)

	for _, event := range l.events {
		if !l.matchesFilter(event, filter) {
			continue
		}
		result = append(result, event)
	}

	// Apply pagination
	if filter.Offset > 0 && filter.Offset < len(result) {
		result = result[filter.Offset:]
	}
	if filter.Limit > 0 && filter.Limit < len(result) {
		result = result[:filter.Limit]
	}

	return result, nil
}

func (l *InMemoryAuditLogger) matchesFilter(event *AuditEvent, filter AuditFilter) bool {
	if len(filter.Types) > 0 {
		found := false
		for _, t := range filter.Types {
			if event.Type == t {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if filter.Actor != "" && event.Actor != filter.Actor {
		return false
	}

	if filter.ResourceType != "" && event.ResourceType != filter.ResourceType {
		return false
	}

	if filter.ResourceID != "" && event.ResourceID != filter.ResourceID {
		return false
	}

	if filter.StartTime != nil && event.Timestamp.Before(*filter.StartTime) {
		return false
	}

	if filter.EndTime != nil && event.Timestamp.After(*filter.EndTime) {
		return false
	}

	return true
}

// AuditService provides audit functionality.
type AuditService struct {
	logger AuditLogger
}

// NewAuditService creates a new audit service.
func NewAuditService(logger AuditLogger) *AuditService {
	return &AuditService{logger: logger}
}

// LogWorkflowStarted logs a workflow start event.
func (s *AuditService) LogWorkflowStarted(ctx context.Context, workflowID, runID, triggeredBy string) error {
	event := NewAuditEvent(AuditEventWorkflowStarted, triggeredBy, "workflow_run", runID, "start").
		WithDetails("workflow_id", workflowID)
	return s.logger.Log(ctx, event)
}

// LogWorkflowCompleted logs a workflow completion event.
func (s *AuditService) LogWorkflowCompleted(ctx context.Context, workflowID, runID string, duration time.Duration) error {
	event := NewAuditEvent(AuditEventWorkflowCompleted, "system", "workflow_run", runID, "complete").
		WithDetails("workflow_id", workflowID).
		WithDetails("duration_ms", duration.Milliseconds())
	return s.logger.Log(ctx, event)
}

// LogWorkflowFailed logs a workflow failure event.
func (s *AuditService) LogWorkflowFailed(ctx context.Context, workflowID, runID, errorMsg string) error {
	event := NewAuditEvent(AuditEventWorkflowFailed, "system", "workflow_run", runID, "fail").
		WithDetails("workflow_id", workflowID).
		WithDetails("error", errorMsg)
	return s.logger.Log(ctx, event)
}

// LogStepExecuted logs a step execution event.
func (s *AuditService) LogStepExecuted(ctx context.Context, runID, stepID, stepName string, tokensUsed int) error {
	event := NewAuditEvent(AuditEventStepExecuted, "system", "step", stepID, "execute").
		WithDetails("run_id", runID).
		WithDetails("step_name", stepName).
		WithDetails("tokens_used", tokensUsed)
	return s.logger.Log(ctx, event)
}

// LogPolicyEvaluated logs a policy evaluation event.
func (s *AuditService) LogPolicyEvaluated(ctx context.Context, runID, policyName string, allowed bool) error {
	event := NewAuditEvent(AuditEventPolicyEvaluated, "system", "workflow_run", runID, "evaluate").
		WithDetails("policy_name", policyName).
		WithDetails("allowed", allowed)
	return s.logger.Log(ctx, event)
}

// LogPolicyViolation logs a policy violation event.
func (s *AuditService) LogPolicyViolation(ctx context.Context, runID, policyName, violation string) error {
	event := NewAuditEvent(AuditEventPolicyViolation, "system", "workflow_run", runID, "violation").
		WithDetails("policy_name", policyName).
		WithDetails("violation", violation)
	return s.logger.Log(ctx, event)
}

// LogApprovalRequested logs an approval request event.
func (s *AuditService) LogApprovalRequested(ctx context.Context, approvalID, runID, requiredBy string) error {
	event := NewAuditEvent(AuditEventApprovalRequested, requiredBy, "approval", approvalID, "request").
		WithDetails("run_id", runID)
	return s.logger.Log(ctx, event)
}

// LogApprovalGranted logs an approval granted event.
func (s *AuditService) LogApprovalGranted(ctx context.Context, approvalID, runID, approvedBy string) error {
	event := NewAuditEvent(AuditEventApprovalGranted, approvedBy, "approval", approvalID, "approve").
		WithDetails("run_id", runID)
	return s.logger.Log(ctx, event)
}

// LogAgentCalled logs an agent invocation event.
func (s *AuditService) LogAgentCalled(ctx context.Context, runID, stepID, agentName, model string, tokensIn, tokensOut int) error {
	event := NewAuditEvent(AuditEventAgentCalled, "system", "step", stepID, "call_agent").
		WithDetails("run_id", runID).
		WithDetails("agent_name", agentName).
		WithDetails("model", model).
		WithDetails("tokens_in", tokensIn).
		WithDetails("tokens_out", tokensOut)
	return s.logger.Log(ctx, event)
}
