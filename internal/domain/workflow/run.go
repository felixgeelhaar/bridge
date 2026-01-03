package workflow

import (
	"time"

	"github.com/felixgeelhaar/bridge/pkg/types"
)

// RunStatus represents the current status of a workflow run.
type RunStatus string

const (
	RunStatusPending          RunStatus = "pending"
	RunStatusPolicyCheck      RunStatus = "policy_check"
	RunStatusAwaitingApproval RunStatus = "awaiting_approval"
	RunStatusExecuting        RunStatus = "executing"
	RunStatusCompleted        RunStatus = "completed"
	RunStatusFailed           RunStatus = "failed"
	RunStatusCancelled        RunStatus = "cancelled"
)

// IsTerminal returns true if the status is a terminal state.
func (s RunStatus) IsTerminal() bool {
	return s == RunStatusCompleted || s == RunStatusFailed || s == RunStatusCancelled
}

// WorkflowRun is the aggregate root for workflow execution.
// It represents a single execution instance of a workflow definition.
type WorkflowRun struct {
	ID              types.RunID
	WorkflowID      types.WorkflowID
	WorkflowName    string
	WorkflowVersion string
	Status          RunStatus
	CurrentStepIdx  int
	Steps           []*StepRun
	Context         map[string]any
	TriggeredBy     string
	TriggerData     map[string]any
	Error           string
	StartedAt       *time.Time
	CompletedAt     *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// NewWorkflowRun creates a new workflow run from a definition.
func NewWorkflowRun(def *WorkflowDefinition, triggeredBy string, triggerData map[string]any) *WorkflowRun {
	now := time.Now()
	run := &WorkflowRun{
		ID:              types.NewRunID(),
		WorkflowID:      def.ID,
		WorkflowName:    def.Name,
		WorkflowVersion: def.Version,
		Status:          RunStatusPending,
		CurrentStepIdx:  0,
		Steps:           make([]*StepRun, 0, len(def.Steps)),
		Context:         make(map[string]any),
		TriggeredBy:     triggeredBy,
		TriggerData:     triggerData,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// Create step runs
	for i, stepDef := range def.Steps {
		run.Steps = append(run.Steps, &StepRun{
			ID:               types.NewStepID(),
			RunID:            run.ID,
			StepIndex:        i,
			Name:             stepDef.Name,
			AgentID:          stepDef.AgentID,
			Status:           StepStatusPending,
			RequiresApproval: stepDef.RequiresApproval,
			Timeout:          stepDef.Timeout,
			MaxRetries:       stepDef.Retries,
			CreatedAt:        now,
		})
	}

	return run
}

// Start begins the workflow execution.
func (r *WorkflowRun) Start() {
	now := time.Now()
	r.Status = RunStatusPolicyCheck
	r.StartedAt = &now
	r.UpdatedAt = now
}

// AwaitApproval sets the run to await approval.
func (r *WorkflowRun) AwaitApproval() {
	r.Status = RunStatusAwaitingApproval
	r.UpdatedAt = time.Now()
}

// Approve approves the workflow and continues execution.
func (r *WorkflowRun) Approve() {
	r.Status = RunStatusExecuting
	r.UpdatedAt = time.Now()
}

// Reject rejects the workflow approval.
func (r *WorkflowRun) Reject(reason string) {
	now := time.Now()
	r.Status = RunStatusCancelled
	r.Error = "approval rejected: " + reason
	r.CompletedAt = &now
	r.UpdatedAt = now
}

// Execute sets the run to executing status.
func (r *WorkflowRun) Execute() {
	r.Status = RunStatusExecuting
	r.UpdatedAt = time.Now()
}

// Complete marks the workflow run as completed.
func (r *WorkflowRun) Complete() {
	now := time.Now()
	r.Status = RunStatusCompleted
	r.CompletedAt = &now
	r.UpdatedAt = now
}

// Fail marks the workflow run as failed.
func (r *WorkflowRun) Fail(err string) {
	now := time.Now()
	r.Status = RunStatusFailed
	r.Error = err
	r.CompletedAt = &now
	r.UpdatedAt = now
}

// Cancel cancels the workflow run.
func (r *WorkflowRun) Cancel(reason string) {
	now := time.Now()
	r.Status = RunStatusCancelled
	r.Error = "cancelled: " + reason
	r.CompletedAt = &now
	r.UpdatedAt = now
}

// CurrentStep returns the current step being executed.
func (r *WorkflowRun) CurrentStep() *StepRun {
	if r.CurrentStepIdx >= 0 && r.CurrentStepIdx < len(r.Steps) {
		return r.Steps[r.CurrentStepIdx]
	}
	return nil
}

// AdvanceStep moves to the next step.
func (r *WorkflowRun) AdvanceStep() bool {
	r.CurrentStepIdx++
	r.UpdatedAt = time.Now()
	return r.CurrentStepIdx < len(r.Steps)
}

// HasMoreSteps returns true if there are more steps to execute.
func (r *WorkflowRun) HasMoreSteps() bool {
	return r.CurrentStepIdx < len(r.Steps)
}

// GetStepByName returns a step run by name.
func (r *WorkflowRun) GetStepByName(name string) *StepRun {
	for _, s := range r.Steps {
		if s.Name == name {
			return s
		}
	}
	return nil
}

// SetContext sets a value in the run context.
func (r *WorkflowRun) SetContext(key string, value any) {
	if r.Context == nil {
		r.Context = make(map[string]any)
	}
	r.Context[key] = value
	r.UpdatedAt = time.Now()
}

// GetContext gets a value from the run context.
func (r *WorkflowRun) GetContext(key string) any {
	if r.Context == nil {
		return nil
	}
	return r.Context[key]
}

// Duration returns the duration of the workflow run.
func (r *WorkflowRun) Duration() time.Duration {
	if r.StartedAt == nil {
		return 0
	}

	end := time.Now()
	if r.CompletedAt != nil {
		end = *r.CompletedAt
	}

	return end.Sub(*r.StartedAt)
}
