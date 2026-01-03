package types

import (
	"github.com/google/uuid"
)

// WorkflowID is a unique identifier for a workflow definition.
type WorkflowID string

// NewWorkflowID generates a new unique WorkflowID.
func NewWorkflowID() WorkflowID {
	return WorkflowID(uuid.New().String())
}

// String returns the string representation of the WorkflowID.
func (id WorkflowID) String() string {
	return string(id)
}

// IsEmpty returns true if the WorkflowID is empty.
func (id WorkflowID) IsEmpty() bool {
	return id == ""
}

// RunID is a unique identifier for a workflow run.
type RunID string

// NewRunID generates a new unique RunID.
func NewRunID() RunID {
	return RunID(uuid.New().String())
}

// String returns the string representation of the RunID.
func (id RunID) String() string {
	return string(id)
}

// IsEmpty returns true if the RunID is empty.
func (id RunID) IsEmpty() bool {
	return id == ""
}

// StepID is a unique identifier for a workflow step.
type StepID string

// NewStepID generates a new unique StepID.
func NewStepID() StepID {
	return StepID(uuid.New().String())
}

// String returns the string representation of the StepID.
func (id StepID) String() string {
	return string(id)
}

// IsEmpty returns true if the StepID is empty.
func (id StepID) IsEmpty() bool {
	return id == ""
}

// AgentID is a unique identifier for an agent.
type AgentID string

// NewAgentID generates a new unique AgentID.
func NewAgentID() AgentID {
	return AgentID(uuid.New().String())
}

// String returns the string representation of the AgentID.
func (id AgentID) String() string {
	return string(id)
}

// IsEmpty returns true if the AgentID is empty.
func (id AgentID) IsEmpty() bool {
	return id == ""
}

// PolicyID is a unique identifier for a policy bundle.
type PolicyID string

// NewPolicyID generates a new unique PolicyID.
func NewPolicyID() PolicyID {
	return PolicyID(uuid.New().String())
}

// String returns the string representation of the PolicyID.
func (id PolicyID) String() string {
	return string(id)
}

// IsEmpty returns true if the PolicyID is empty.
func (id PolicyID) IsEmpty() bool {
	return id == ""
}

// ApprovalID is a unique identifier for an approval request.
type ApprovalID string

// NewApprovalID generates a new unique ApprovalID.
func NewApprovalID() ApprovalID {
	return ApprovalID(uuid.New().String())
}

// String returns the string representation of the ApprovalID.
func (id ApprovalID) String() string {
	return string(id)
}

// IsEmpty returns true if the ApprovalID is empty.
func (id ApprovalID) IsEmpty() bool {
	return id == ""
}
