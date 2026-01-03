package types

import (
	"errors"
	"testing"
)

func TestDomainErrors(t *testing.T) {
	// Test that all domain errors are defined and non-nil
	domainErrors := []struct {
		name string
		err  error
	}{
		{"ErrWorkflowNotFound", ErrWorkflowNotFound},
		{"ErrWorkflowAlreadyExists", ErrWorkflowAlreadyExists},
		{"ErrWorkflowInvalid", ErrWorkflowInvalid},
		{"ErrRunNotFound", ErrRunNotFound},
		{"ErrRunAlreadyStarted", ErrRunAlreadyStarted},
		{"ErrRunCompleted", ErrRunCompleted},
		{"ErrRunCancelled", ErrRunCancelled},
		{"ErrStepNotFound", ErrStepNotFound},
		{"ErrStepFailed", ErrStepFailed},
		{"ErrStepTimeout", ErrStepTimeout},
		{"ErrPolicyNotFound", ErrPolicyNotFound},
		{"ErrPolicyViolation", ErrPolicyViolation},
		{"ErrPolicyInvalid", ErrPolicyInvalid},
		{"ErrApprovalNotFound", ErrApprovalNotFound},
		{"ErrApprovalRequired", ErrApprovalRequired},
		{"ErrApprovalRejected", ErrApprovalRejected},
		{"ErrApprovalExpired", ErrApprovalExpired},
		{"ErrApprovalPending", ErrApprovalPending},
		{"ErrAgentNotFound", ErrAgentNotFound},
		{"ErrAgentUnavailable", ErrAgentUnavailable},
		{"ErrAgentTimeout", ErrAgentTimeout},
		{"ErrLLMProviderNotFound", ErrLLMProviderNotFound},
		{"ErrLLMRateLimited", ErrLLMRateLimited},
		{"ErrLLMContextTooLong", ErrLLMContextTooLong},
		{"ErrMCPToolNotFound", ErrMCPToolNotFound},
		{"ErrMCPToolForbidden", ErrMCPToolForbidden},
	}

	for _, tt := range domainErrors {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Errorf("%s should not be nil", tt.name)
			}
			if tt.err.Error() == "" {
				t.Errorf("%s.Error() should not be empty", tt.name)
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	err := NewValidationError("email", "must be a valid email address")

	if err.Field != "email" {
		t.Errorf("Field = %v, want email", err.Field)
	}
	if err.Message != "must be a valid email address" {
		t.Errorf("Message = %v, want 'must be a valid email address'", err.Message)
	}

	// Test Error() method
	errStr := err.Error()
	if errStr == "" {
		t.Error("Error() should not return empty string")
	}
	// Should contain field name and message
	if !containsString(errStr, "email") {
		t.Error("Error() should contain field name")
	}
	if !containsString(errStr, "must be a valid email address") {
		t.Error("Error() should contain message")
	}
}

func TestDomainError(t *testing.T) {
	baseErr := errors.New("connection refused")
	domainErr := NewDomainError("GetWorkflow", ErrWorkflowNotFound, baseErr)

	if domainErr.Op != "GetWorkflow" {
		t.Errorf("Op = %v, want GetWorkflow", domainErr.Op)
	}
	if domainErr.Kind != ErrWorkflowNotFound {
		t.Errorf("Kind = %v, want ErrWorkflowNotFound", domainErr.Kind)
	}
	if domainErr.Err != baseErr {
		t.Error("Err should be the original error")
	}

	// Test Error() method
	errStr := domainErr.Error()
	if errStr == "" {
		t.Error("Error() should not return empty string")
	}
	if !containsString(errStr, "GetWorkflow") {
		t.Error("Error() should contain operation")
	}
}

func TestDomainError_WithContext(t *testing.T) {
	domainErr := NewDomainError("CreateRun", ErrRunAlreadyStarted, nil).
		WithContext("run_id", "run-123").
		WithContext("workflow_id", "wf-456")

	if domainErr.Context["run_id"] != "run-123" {
		t.Error("Context should contain run_id")
	}
	if domainErr.Context["workflow_id"] != "wf-456" {
		t.Error("Context should contain workflow_id")
	}
}

func TestDomainError_Unwrap(t *testing.T) {
	baseErr := errors.New("underlying error")
	domainErr := NewDomainError("TestOp", ErrStepFailed, baseErr)

	unwrapped := domainErr.Unwrap()
	if unwrapped != baseErr {
		t.Error("Unwrap() should return the underlying error")
	}
}

func TestDomainError_Is(t *testing.T) {
	domainErr := NewDomainError("TestOp", ErrPolicyViolation, nil)

	if !errors.Is(domainErr, ErrPolicyViolation) {
		t.Error("errors.Is should return true for matching Kind")
	}
	if errors.Is(domainErr, ErrWorkflowNotFound) {
		t.Error("errors.Is should return false for non-matching Kind")
	}
}

func TestDomainError_NoUnderlyingError(t *testing.T) {
	domainErr := NewDomainError("TestOp", ErrApprovalRequired, nil)

	errStr := domainErr.Error()
	if errStr == "" {
		t.Error("Error() should not return empty string even without underlying error")
	}
	if !containsString(errStr, "TestOp") {
		t.Error("Error() should contain operation")
	}
}

func TestIsTransient(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"ErrAgentTimeout", ErrAgentTimeout, true},
		{"ErrLLMRateLimited", ErrLLMRateLimited, true},
		{"ErrAgentUnavailable", ErrAgentUnavailable, true},
		{"ErrWorkflowNotFound", ErrWorkflowNotFound, false},
		{"ErrPolicyViolation", ErrPolicyViolation, false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsTransient(tt.err)
			if got != tt.want {
				t.Errorf("IsTransient(%v) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestIsTransient_WrappedError(t *testing.T) {
	// Test that IsTransient works with wrapped errors
	wrappedErr := NewDomainError("TestOp", ErrAgentTimeout, nil)

	if !IsTransient(wrappedErr) {
		t.Error("IsTransient should return true for wrapped transient error")
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"ErrWorkflowNotFound", ErrWorkflowNotFound, true},
		{"ErrRunNotFound", ErrRunNotFound, true},
		{"ErrStepNotFound", ErrStepNotFound, true},
		{"ErrPolicyNotFound", ErrPolicyNotFound, true},
		{"ErrApprovalNotFound", ErrApprovalNotFound, true},
		{"ErrAgentNotFound", ErrAgentNotFound, true},
		{"ErrPolicyViolation", ErrPolicyViolation, false},
		{"ErrAgentTimeout", ErrAgentTimeout, false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNotFound(tt.err)
			if got != tt.want {
				t.Errorf("IsNotFound(%v) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestIsNotFound_WrappedError(t *testing.T) {
	// Test that IsNotFound works with wrapped errors
	wrappedErr := NewDomainError("TestOp", ErrWorkflowNotFound, nil)

	if !IsNotFound(wrappedErr) {
		t.Error("IsNotFound should return true for wrapped not-found error")
	}
}

func TestErrorMessages(t *testing.T) {
	// Test that error messages are descriptive
	errMessages := map[error]string{
		ErrWorkflowNotFound:      "workflow not found",
		ErrWorkflowAlreadyExists: "workflow already exists",
		ErrPolicyViolation:       "policy violation",
		ErrApprovalRequired:      "approval required",
		ErrLLMRateLimited:        "LLM rate limited",
	}

	for err, expected := range errMessages {
		if err.Error() != expected {
			t.Errorf("Error message for %v = %q, want %q", err, err.Error(), expected)
		}
	}
}

// Helper function
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
