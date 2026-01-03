package types

import (
	"errors"
	"fmt"
)

// Domain errors
var (
	// Workflow errors
	ErrWorkflowNotFound      = errors.New("workflow not found")
	ErrWorkflowAlreadyExists = errors.New("workflow already exists")
	ErrWorkflowInvalid       = errors.New("workflow definition is invalid")

	// Run errors
	ErrRunNotFound       = errors.New("workflow run not found")
	ErrRunAlreadyStarted = errors.New("workflow run already started")
	ErrRunCompleted      = errors.New("workflow run already completed")
	ErrRunCancelled      = errors.New("workflow run was cancelled")

	// Step errors
	ErrStepNotFound = errors.New("step not found")
	ErrStepFailed   = errors.New("step execution failed")
	ErrStepTimeout  = errors.New("step execution timed out")

	// Policy errors
	ErrPolicyNotFound  = errors.New("policy not found")
	ErrPolicyViolation = errors.New("policy violation")
	ErrPolicyInvalid   = errors.New("policy definition is invalid")

	// Approval errors
	ErrApprovalNotFound  = errors.New("approval not found")
	ErrApprovalRequired  = errors.New("approval required")
	ErrApprovalRejected  = errors.New("approval rejected")
	ErrApprovalExpired   = errors.New("approval expired")
	ErrApprovalPending   = errors.New("approval pending")

	// Agent errors
	ErrAgentNotFound    = errors.New("agent not found")
	ErrAgentUnavailable = errors.New("agent unavailable")
	ErrAgentTimeout     = errors.New("agent call timed out")

	// LLM errors
	ErrLLMProviderNotFound = errors.New("LLM provider not found")
	ErrLLMRateLimited      = errors.New("LLM rate limited")
	ErrLLMContextTooLong   = errors.New("context too long for model")

	// MCP errors
	ErrMCPToolNotFound  = errors.New("MCP tool not found")
	ErrMCPToolForbidden = errors.New("MCP tool forbidden by policy")
)

// ValidationError represents a validation error with details.
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error: %s - %s", e.Field, e.Message)
}

// NewValidationError creates a new validation error.
func NewValidationError(field, message string) ValidationError {
	return ValidationError{Field: field, Message: message}
}

// DomainError wraps an error with additional context.
type DomainError struct {
	Op      string // Operation that failed
	Kind    error  // Category of error
	Err     error  // Underlying error
	Context map[string]any
}

func (e *DomainError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v: %v", e.Op, e.Kind, e.Err)
	}
	return fmt.Sprintf("%s: %v", e.Op, e.Kind)
}

func (e *DomainError) Unwrap() error {
	return e.Err
}

func (e *DomainError) Is(target error) bool {
	return errors.Is(e.Kind, target)
}

// NewDomainError creates a new domain error.
func NewDomainError(op string, kind error, err error) *DomainError {
	return &DomainError{
		Op:   op,
		Kind: kind,
		Err:  err,
	}
}

// WithContext adds context to a domain error.
func (e *DomainError) WithContext(key string, value any) *DomainError {
	if e.Context == nil {
		e.Context = make(map[string]any)
	}
	e.Context[key] = value
	return e
}

// IsTransient returns true if the error is transient and can be retried.
func IsTransient(err error) bool {
	return errors.Is(err, ErrAgentTimeout) ||
		errors.Is(err, ErrLLMRateLimited) ||
		errors.Is(err, ErrAgentUnavailable)
}

// IsNotFound returns true if the error is a not found error.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrWorkflowNotFound) ||
		errors.Is(err, ErrRunNotFound) ||
		errors.Is(err, ErrStepNotFound) ||
		errors.Is(err, ErrPolicyNotFound) ||
		errors.Is(err, ErrApprovalNotFound) ||
		errors.Is(err, ErrAgentNotFound)
}
