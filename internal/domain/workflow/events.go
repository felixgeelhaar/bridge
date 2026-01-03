package workflow

import (
	"time"

	"github.com/felixgeelhaar/bridge/pkg/types"
	"github.com/google/uuid"
)

// Event is the base interface for all domain events.
type Event interface {
	EventID() string
	EventType() string
	OccurredAt() time.Time
	AggregateID() string
}

// BaseEvent contains common event fields.
type BaseEvent struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Aggregate string    `json:"aggregate_id"`
}

func (e BaseEvent) EventID() string      { return e.ID }
func (e BaseEvent) EventType() string    { return e.Type }
func (e BaseEvent) OccurredAt() time.Time { return e.Timestamp }
func (e BaseEvent) AggregateID() string  { return e.Aggregate }

func newBaseEvent(eventType, aggregateID string) BaseEvent {
	return BaseEvent{
		ID:        uuid.New().String(),
		Type:      eventType,
		Timestamp: time.Now(),
		Aggregate: aggregateID,
	}
}

// WorkflowCreatedEvent is emitted when a new workflow definition is created.
type WorkflowCreatedEvent struct {
	BaseEvent
	WorkflowID types.WorkflowID `json:"workflow_id"`
	Name       string           `json:"name"`
	Version    string           `json:"version"`
}

func NewWorkflowCreatedEvent(def *WorkflowDefinition) *WorkflowCreatedEvent {
	return &WorkflowCreatedEvent{
		BaseEvent:  newBaseEvent("workflow.created", def.ID.String()),
		WorkflowID: def.ID,
		Name:       def.Name,
		Version:    def.Version,
	}
}

// RunStartedEvent is emitted when a workflow run is started.
type RunStartedEvent struct {
	BaseEvent
	RunID       types.RunID      `json:"run_id"`
	WorkflowID  types.WorkflowID `json:"workflow_id"`
	TriggeredBy string           `json:"triggered_by"`
}

func NewRunStartedEvent(run *WorkflowRun) *RunStartedEvent {
	return &RunStartedEvent{
		BaseEvent:   newBaseEvent("run.started", run.ID.String()),
		RunID:       run.ID,
		WorkflowID:  run.WorkflowID,
		TriggeredBy: run.TriggeredBy,
	}
}

// RunCompletedEvent is emitted when a workflow run completes successfully.
type RunCompletedEvent struct {
	BaseEvent
	RunID       types.RunID      `json:"run_id"`
	WorkflowID  types.WorkflowID `json:"workflow_id"`
	Duration    time.Duration    `json:"duration"`
	StepsCount  int              `json:"steps_count"`
	TotalTokens int              `json:"total_tokens"`
}

func NewRunCompletedEvent(run *WorkflowRun) *RunCompletedEvent {
	totalTokens := 0
	for _, step := range run.Steps {
		totalTokens += step.TokensIn + step.TokensOut
	}
	return &RunCompletedEvent{
		BaseEvent:   newBaseEvent("run.completed", run.ID.String()),
		RunID:       run.ID,
		WorkflowID:  run.WorkflowID,
		Duration:    run.Duration(),
		StepsCount:  len(run.Steps),
		TotalTokens: totalTokens,
	}
}

// RunFailedEvent is emitted when a workflow run fails.
type RunFailedEvent struct {
	BaseEvent
	RunID      types.RunID      `json:"run_id"`
	WorkflowID types.WorkflowID `json:"workflow_id"`
	Error      string           `json:"error"`
	FailedStep string           `json:"failed_step,omitempty"`
}

func NewRunFailedEvent(run *WorkflowRun, failedStep string) *RunFailedEvent {
	return &RunFailedEvent{
		BaseEvent:  newBaseEvent("run.failed", run.ID.String()),
		RunID:      run.ID,
		WorkflowID: run.WorkflowID,
		Error:      run.Error,
		FailedStep: failedStep,
	}
}

// RunCancelledEvent is emitted when a workflow run is cancelled.
type RunCancelledEvent struct {
	BaseEvent
	RunID      types.RunID      `json:"run_id"`
	WorkflowID types.WorkflowID `json:"workflow_id"`
	Reason     string           `json:"reason"`
}

func NewRunCancelledEvent(run *WorkflowRun) *RunCancelledEvent {
	return &RunCancelledEvent{
		BaseEvent:  newBaseEvent("run.cancelled", run.ID.String()),
		RunID:      run.ID,
		WorkflowID: run.WorkflowID,
		Reason:     run.Error,
	}
}

// StepStartedEvent is emitted when a step begins execution.
type StepStartedEvent struct {
	BaseEvent
	RunID   types.RunID  `json:"run_id"`
	StepID  types.StepID `json:"step_id"`
	Name    string       `json:"name"`
	AgentID string       `json:"agent_id"`
}

func NewStepStartedEvent(runID types.RunID, step *StepRun) *StepStartedEvent {
	return &StepStartedEvent{
		BaseEvent: newBaseEvent("step.started", step.ID.String()),
		RunID:     runID,
		StepID:    step.ID,
		Name:      step.Name,
		AgentID:   step.AgentID,
	}
}

// StepCompletedEvent is emitted when a step completes successfully.
type StepCompletedEvent struct {
	BaseEvent
	RunID     types.RunID   `json:"run_id"`
	StepID    types.StepID  `json:"step_id"`
	Name      string        `json:"name"`
	Duration  time.Duration `json:"duration"`
	TokensIn  int           `json:"tokens_in"`
	TokensOut int           `json:"tokens_out"`
}

func NewStepCompletedEvent(runID types.RunID, step *StepRun) *StepCompletedEvent {
	return &StepCompletedEvent{
		BaseEvent: newBaseEvent("step.completed", step.ID.String()),
		RunID:     runID,
		StepID:    step.ID,
		Name:      step.Name,
		Duration:  step.Duration(),
		TokensIn:  step.TokensIn,
		TokensOut: step.TokensOut,
	}
}

// StepFailedEvent is emitted when a step fails.
type StepFailedEvent struct {
	BaseEvent
	RunID      types.RunID  `json:"run_id"`
	StepID     types.StepID `json:"step_id"`
	Name       string       `json:"name"`
	Error      string       `json:"error"`
	RetryCount int          `json:"retry_count"`
	CanRetry   bool         `json:"can_retry"`
}

func NewStepFailedEvent(runID types.RunID, step *StepRun) *StepFailedEvent {
	return &StepFailedEvent{
		BaseEvent:  newBaseEvent("step.failed", step.ID.String()),
		RunID:      runID,
		StepID:     step.ID,
		Name:       step.Name,
		Error:      step.Error,
		RetryCount: step.RetryCount,
		CanRetry:   step.CanRetry(),
	}
}

// ApprovalRequestedEvent is emitted when a workflow requires approval.
type ApprovalRequestedEvent struct {
	BaseEvent
	RunID        types.RunID      `json:"run_id"`
	WorkflowID   types.WorkflowID `json:"workflow_id"`
	WorkflowName string           `json:"workflow_name"`
	StepName     string           `json:"step_name,omitempty"`
}

func NewApprovalRequestedEvent(run *WorkflowRun, stepName string) *ApprovalRequestedEvent {
	return &ApprovalRequestedEvent{
		BaseEvent:    newBaseEvent("approval.requested", run.ID.String()),
		RunID:        run.ID,
		WorkflowID:   run.WorkflowID,
		WorkflowName: run.WorkflowName,
		StepName:     stepName,
	}
}

// ApprovalGrantedEvent is emitted when a workflow approval is granted.
type ApprovalGrantedEvent struct {
	BaseEvent
	RunID      types.RunID `json:"run_id"`
	ApprovedBy string      `json:"approved_by"`
}

func NewApprovalGrantedEvent(runID types.RunID, approvedBy string) *ApprovalGrantedEvent {
	return &ApprovalGrantedEvent{
		BaseEvent:  newBaseEvent("approval.granted", runID.String()),
		RunID:      runID,
		ApprovedBy: approvedBy,
	}
}

// PolicyViolationEvent is emitted when a policy is violated.
type PolicyViolationEvent struct {
	BaseEvent
	RunID      types.RunID `json:"run_id"`
	PolicyName string      `json:"policy_name"`
	Violations []string    `json:"violations"`
}

func NewPolicyViolationEvent(runID types.RunID, policyName string, violations []string) *PolicyViolationEvent {
	return &PolicyViolationEvent{
		BaseEvent:  newBaseEvent("policy.violation", runID.String()),
		RunID:      runID,
		PolicyName: policyName,
		Violations: violations,
	}
}
