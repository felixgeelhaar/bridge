package types

import (
	"testing"
)

func TestWorkflowID(t *testing.T) {
	// Test NewWorkflowID generates unique IDs
	id1 := NewWorkflowID()
	id2 := NewWorkflowID()

	if id1 == "" {
		t.Error("NewWorkflowID should not return empty string")
	}
	if id1 == id2 {
		t.Error("NewWorkflowID should generate unique IDs")
	}

	// Test String method
	if id1.String() != string(id1) {
		t.Error("String() should return the ID as string")
	}

	// Test IsEmpty
	var emptyID WorkflowID
	if !emptyID.IsEmpty() {
		t.Error("Empty WorkflowID should return true for IsEmpty()")
	}
	if id1.IsEmpty() {
		t.Error("Non-empty WorkflowID should return false for IsEmpty()")
	}
}

func TestRunID(t *testing.T) {
	// Test NewRunID generates unique IDs
	id1 := NewRunID()
	id2 := NewRunID()

	if id1 == "" {
		t.Error("NewRunID should not return empty string")
	}
	if id1 == id2 {
		t.Error("NewRunID should generate unique IDs")
	}

	// Test String method
	if id1.String() != string(id1) {
		t.Error("String() should return the ID as string")
	}

	// Test IsEmpty
	var emptyID RunID
	if !emptyID.IsEmpty() {
		t.Error("Empty RunID should return true for IsEmpty()")
	}
	if id1.IsEmpty() {
		t.Error("Non-empty RunID should return false for IsEmpty()")
	}
}

func TestStepID(t *testing.T) {
	// Test NewStepID generates unique IDs
	id1 := NewStepID()
	id2 := NewStepID()

	if id1 == "" {
		t.Error("NewStepID should not return empty string")
	}
	if id1 == id2 {
		t.Error("NewStepID should generate unique IDs")
	}

	// Test String method
	if id1.String() != string(id1) {
		t.Error("String() should return the ID as string")
	}

	// Test IsEmpty
	var emptyID StepID
	if !emptyID.IsEmpty() {
		t.Error("Empty StepID should return true for IsEmpty()")
	}
	if id1.IsEmpty() {
		t.Error("Non-empty StepID should return false for IsEmpty()")
	}
}

func TestAgentID(t *testing.T) {
	// Test NewAgentID generates unique IDs
	id1 := NewAgentID()
	id2 := NewAgentID()

	if id1 == "" {
		t.Error("NewAgentID should not return empty string")
	}
	if id1 == id2 {
		t.Error("NewAgentID should generate unique IDs")
	}

	// Test String method
	if id1.String() != string(id1) {
		t.Error("String() should return the ID as string")
	}

	// Test IsEmpty
	var emptyID AgentID
	if !emptyID.IsEmpty() {
		t.Error("Empty AgentID should return true for IsEmpty()")
	}
	if id1.IsEmpty() {
		t.Error("Non-empty AgentID should return false for IsEmpty()")
	}
}

func TestPolicyID(t *testing.T) {
	// Test NewPolicyID generates unique IDs
	id1 := NewPolicyID()
	id2 := NewPolicyID()

	if id1 == "" {
		t.Error("NewPolicyID should not return empty string")
	}
	if id1 == id2 {
		t.Error("NewPolicyID should generate unique IDs")
	}

	// Test String method
	if id1.String() != string(id1) {
		t.Error("String() should return the ID as string")
	}

	// Test IsEmpty
	var emptyID PolicyID
	if !emptyID.IsEmpty() {
		t.Error("Empty PolicyID should return true for IsEmpty()")
	}
	if id1.IsEmpty() {
		t.Error("Non-empty PolicyID should return false for IsEmpty()")
	}
}

func TestApprovalID(t *testing.T) {
	// Test NewApprovalID generates unique IDs
	id1 := NewApprovalID()
	id2 := NewApprovalID()

	if id1 == "" {
		t.Error("NewApprovalID should not return empty string")
	}
	if id1 == id2 {
		t.Error("NewApprovalID should generate unique IDs")
	}

	// Test String method
	if id1.String() != string(id1) {
		t.Error("String() should return the ID as string")
	}

	// Test IsEmpty
	var emptyID ApprovalID
	if !emptyID.IsEmpty() {
		t.Error("Empty ApprovalID should return true for IsEmpty()")
	}
	if id1.IsEmpty() {
		t.Error("Non-empty ApprovalID should return false for IsEmpty()")
	}
}

func TestIDFormats(t *testing.T) {
	// All IDs should be valid UUIDs (36 characters with hyphens)
	tests := []struct {
		name string
		id   string
	}{
		{"WorkflowID", NewWorkflowID().String()},
		{"RunID", NewRunID().String()},
		{"StepID", NewStepID().String()},
		{"AgentID", NewAgentID().String()},
		{"PolicyID", NewPolicyID().String()},
		{"ApprovalID", NewApprovalID().String()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.id) != 36 {
				t.Errorf("%s length = %d, want 36 (UUID format)", tt.name, len(tt.id))
			}
			// Check UUID format: 8-4-4-4-12
			if tt.id[8] != '-' || tt.id[13] != '-' || tt.id[18] != '-' || tt.id[23] != '-' {
				t.Errorf("%s is not in valid UUID format: %s", tt.name, tt.id)
			}
		})
	}
}

func TestIDTypeConversion(t *testing.T) {
	// Test that IDs can be converted from strings
	workflowID := WorkflowID("test-workflow-id")
	if workflowID.String() != "test-workflow-id" {
		t.Error("WorkflowID string conversion failed")
	}

	runID := RunID("test-run-id")
	if runID.String() != "test-run-id" {
		t.Error("RunID string conversion failed")
	}

	stepID := StepID("test-step-id")
	if stepID.String() != "test-step-id" {
		t.Error("StepID string conversion failed")
	}

	agentID := AgentID("test-agent-id")
	if agentID.String() != "test-agent-id" {
		t.Error("AgentID string conversion failed")
	}

	policyID := PolicyID("test-policy-id")
	if policyID.String() != "test-policy-id" {
		t.Error("PolicyID string conversion failed")
	}

	approvalID := ApprovalID("test-approval-id")
	if approvalID.String() != "test-approval-id" {
		t.Error("ApprovalID string conversion failed")
	}
}
