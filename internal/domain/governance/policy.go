package governance

import (
	"context"
	"time"

	"github.com/felixgeelhaar/bridge/pkg/types"
)

// PolicyBundle represents a collection of policies.
type PolicyBundle struct {
	ID          types.PolicyID
	Name        string
	Version     string
	Description string
	Rules       []PolicyRule
	Checksum    string
	Active      bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// PolicyRule represents a single policy rule.
type PolicyRule struct {
	Name        string
	Description string
	Rego        string // Rego policy code
	Severity    Severity
	Enabled     bool
}

// Severity represents the severity of a policy violation.
type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
	SeverityCritical Severity = "critical"
)

// PolicyInput provides context for policy evaluation.
type PolicyInput struct {
	WorkflowID   string            `json:"workflow_id"`
	WorkflowName string            `json:"workflow_name"`
	RunID        string            `json:"run_id"`
	StepName     string            `json:"step_name,omitempty"`
	AgentID      string            `json:"agent_id,omitempty"`
	AgentName    string            `json:"agent_name,omitempty"`
	Capabilities []string          `json:"capabilities,omitempty"`
	Context      map[string]any    `json:"context,omitempty"`
	Metadata     map[string]any    `json:"metadata,omitempty"`
}

// PolicyResult contains the result of policy evaluation.
type PolicyResult struct {
	Allowed          bool              `json:"allowed"`
	RequiresApproval bool              `json:"requires_approval"`
	Violations       []Violation       `json:"violations,omitempty"`
	Warnings         []Violation       `json:"warnings,omitempty"`
	Metadata         map[string]any    `json:"metadata,omitempty"`
}

// Violation represents a policy violation.
type Violation struct {
	Rule     string   `json:"rule"`
	Message  string   `json:"message"`
	Severity Severity `json:"severity"`
	Details  map[string]any `json:"details,omitempty"`
}

// IsBlocking returns true if the result blocks execution.
func (r *PolicyResult) IsBlocking() bool {
	if !r.Allowed {
		return true
	}

	for _, v := range r.Violations {
		if v.Severity == SeverityError || v.Severity == SeverityCritical {
			return true
		}
	}

	return false
}

// Evaluator evaluates policies against inputs.
type Evaluator interface {
	// Evaluate evaluates a policy bundle against an input.
	Evaluate(ctx context.Context, bundle *PolicyBundle, input *PolicyInput) (*PolicyResult, error)

	// EvaluateAll evaluates all active policy bundles.
	EvaluateAll(ctx context.Context, input *PolicyInput) (*PolicyResult, error)
}

// Repository provides persistence for policy bundles.
type Repository interface {
	Create(ctx context.Context, bundle *PolicyBundle) error
	Get(ctx context.Context, id types.PolicyID) (*PolicyBundle, error)
	GetByName(ctx context.Context, name string) (*PolicyBundle, error)
	List(ctx context.Context, activeOnly bool) ([]*PolicyBundle, error)
	Update(ctx context.Context, bundle *PolicyBundle) error
	Delete(ctx context.Context, id types.PolicyID) error
}

// NewPolicyBundle creates a new policy bundle.
func NewPolicyBundle(name, version, description string) *PolicyBundle {
	now := time.Now()
	return &PolicyBundle{
		ID:          types.NewPolicyID(),
		Name:        name,
		Version:     version,
		Description: description,
		Rules:       make([]PolicyRule, 0),
		Active:      true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// AddRule adds a rule to the bundle.
func (b *PolicyBundle) AddRule(rule PolicyRule) {
	b.Rules = append(b.Rules, rule)
	b.UpdatedAt = time.Now()
}

// DefaultPolicyBundle returns a default policy bundle with common rules.
func DefaultPolicyBundle() *PolicyBundle {
	bundle := NewPolicyBundle("default", "1.0.0", "Default Bridge policies")

	bundle.AddRule(PolicyRule{
		Name:        "require_approval_for_writes",
		Description: "Require approval for file write operations",
		Rego: `
package bridge.policy

default requires_approval = false

requires_approval {
    input.capabilities[_] == "file-write"
}
`,
		Severity: SeverityWarning,
		Enabled:  true,
	})

	bundle.AddRule(PolicyRule{
		Name:        "block_sensitive_files",
		Description: "Block access to sensitive files",
		Rego: `
package bridge.policy

default allowed = true

allowed = false {
    contains(input.context.path, ".env")
}

allowed = false {
    contains(input.context.path, "secrets")
}

violation[msg] {
    not allowed
    msg := sprintf("Access to sensitive file blocked: %s", [input.context.path])
}
`,
		Severity: SeverityCritical,
		Enabled:  true,
	})

	bundle.AddRule(PolicyRule{
		Name:        "max_token_limit",
		Description: "Enforce maximum token usage per request",
		Rego: `
package bridge.policy

default allowed = true

allowed = false {
    input.context.max_tokens > 100000
}

violation[msg] {
    not allowed
    msg := sprintf("Token limit exceeded: %d > 100000", [input.context.max_tokens])
}
`,
		Severity: SeverityError,
		Enabled:  true,
	})

	return bundle
}
