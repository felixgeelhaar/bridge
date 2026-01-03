package workflow

import (
	"time"

	"github.com/felixgeelhaar/bridge/pkg/types"
)

// StepStatus represents the current status of a step execution.
type StepStatus string

const (
	StepStatusPending   StepStatus = "pending"
	StepStatusRunning   StepStatus = "running"
	StepStatusCompleted StepStatus = "completed"
	StepStatusFailed    StepStatus = "failed"
	StepStatusSkipped   StepStatus = "skipped"
)

// IsTerminal returns true if the status is a terminal state.
func (s StepStatus) IsTerminal() bool {
	return s == StepStatusCompleted || s == StepStatusFailed || s == StepStatusSkipped
}

// StepRun represents a single step execution within a workflow run.
type StepRun struct {
	ID               types.StepID
	RunID            types.RunID
	StepIndex        int
	Name             string
	AgentID          string
	Status           StepStatus
	Input            map[string]any
	Output           map[string]any
	RequiresApproval bool
	Timeout          time.Duration
	MaxRetries       int
	RetryCount       int
	Error            string
	TokensIn         int
	TokensOut        int
	StartedAt        *time.Time
	CompletedAt      *time.Time
	CreatedAt        time.Time
}

// Start begins the step execution.
func (s *StepRun) Start(input map[string]any) {
	now := time.Now()
	s.Status = StepStatusRunning
	s.Input = input
	s.StartedAt = &now
}

// Complete marks the step as completed with output.
func (s *StepRun) Complete(output map[string]any, tokensIn, tokensOut int) {
	now := time.Now()
	s.Status = StepStatusCompleted
	s.Output = output
	s.TokensIn = tokensIn
	s.TokensOut = tokensOut
	s.CompletedAt = &now
}

// Fail marks the step as failed with an error.
func (s *StepRun) Fail(err string) {
	now := time.Now()
	s.Status = StepStatusFailed
	s.Error = err
	s.CompletedAt = &now
}

// Skip marks the step as skipped.
func (s *StepRun) Skip(reason string) {
	now := time.Now()
	s.Status = StepStatusSkipped
	s.Error = reason
	s.CompletedAt = &now
}

// CanRetry returns true if the step can be retried.
func (s *StepRun) CanRetry() bool {
	return s.Status == StepStatusFailed && s.RetryCount < s.MaxRetries
}

// IncrementRetry increments the retry counter and resets status.
func (s *StepRun) IncrementRetry() {
	s.RetryCount++
	s.Status = StepStatusPending
	s.Error = ""
	s.StartedAt = nil
	s.CompletedAt = nil
}

// Duration returns the duration of the step execution.
func (s *StepRun) Duration() time.Duration {
	if s.StartedAt == nil {
		return 0
	}

	end := time.Now()
	if s.CompletedAt != nil {
		end = *s.CompletedAt
	}

	return end.Sub(*s.StartedAt)
}

// IsRunning returns true if the step is currently running.
func (s *StepRun) IsRunning() bool {
	return s.Status == StepStatusRunning
}

// IsPending returns true if the step is pending.
func (s *StepRun) IsPending() bool {
	return s.Status == StepStatusPending
}

// TokenUsage represents the token usage for an LLM call.
type TokenUsage struct {
	Input  int
	Output int
	Total  int
}

// StepResult represents the result of a step execution.
type StepResult struct {
	StepID    types.StepID
	Status    StepStatus
	Output    map[string]any
	Tokens    TokenUsage
	Duration  time.Duration
	Error     string
}

// NewStepResult creates a step result from a step run.
func NewStepResult(step *StepRun) *StepResult {
	return &StepResult{
		StepID:   step.ID,
		Status:   step.Status,
		Output:   step.Output,
		Tokens:   TokenUsage{Input: step.TokensIn, Output: step.TokensOut, Total: step.TokensIn + step.TokensOut},
		Duration: step.Duration(),
		Error:    step.Error,
	}
}
