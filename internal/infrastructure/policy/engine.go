package policy

import (
	"context"
	"fmt"
	"strings"

	"github.com/felixgeelhaar/bolt"
	"github.com/felixgeelhaar/bridge/internal/domain/governance"
	"github.com/open-policy-agent/opa/v1/rego"
)

// Engine wraps OPA for policy evaluation.
type Engine struct {
	logger  *bolt.Logger
	bundles []*governance.PolicyBundle
}

// NewEngine creates a new policy engine.
func NewEngine(logger *bolt.Logger) *Engine {
	return &Engine{
		logger:  logger,
		bundles: make([]*governance.PolicyBundle, 0),
	}
}

// LoadBundle loads a policy bundle into the engine.
func (e *Engine) LoadBundle(bundle *governance.PolicyBundle) {
	e.bundles = append(e.bundles, bundle)
	e.logger.Info().
		Str("bundle", bundle.Name).
		Str("version", bundle.Version).
		Int("rules", len(bundle.Rules)).
		Msg("Policy bundle loaded")
}

// Evaluate evaluates a policy bundle against an input.
func (e *Engine) Evaluate(ctx context.Context, bundle *governance.PolicyBundle, input *governance.PolicyInput) (*governance.PolicyResult, error) {
	result := &governance.PolicyResult{
		Allowed:    true,
		Violations: make([]governance.Violation, 0),
		Warnings:   make([]governance.Violation, 0),
		Metadata:   make(map[string]any),
	}

	for _, rule := range bundle.Rules {
		if !rule.Enabled {
			continue
		}

		ruleResult, err := e.evaluateRule(ctx, rule, input)
		if err != nil {
			e.logger.Error().
				Err(err).
				Str("rule", rule.Name).
				Msg("Failed to evaluate rule")
			continue
		}

		// Merge results
		if !ruleResult.Allowed {
			result.Allowed = false
		}
		if ruleResult.RequiresApproval {
			result.RequiresApproval = true
		}
		result.Violations = append(result.Violations, ruleResult.Violations...)
		result.Warnings = append(result.Warnings, ruleResult.Warnings...)
	}

	return result, nil
}

// EvaluateAll evaluates all active policy bundles.
func (e *Engine) EvaluateAll(ctx context.Context, input *governance.PolicyInput) (*governance.PolicyResult, error) {
	result := &governance.PolicyResult{
		Allowed:    true,
		Violations: make([]governance.Violation, 0),
		Warnings:   make([]governance.Violation, 0),
		Metadata:   make(map[string]any),
	}

	for _, bundle := range e.bundles {
		if !bundle.Active {
			continue
		}

		bundleResult, err := e.Evaluate(ctx, bundle, input)
		if err != nil {
			return nil, err
		}

		// Merge results
		if !bundleResult.Allowed {
			result.Allowed = false
		}
		if bundleResult.RequiresApproval {
			result.RequiresApproval = true
		}
		result.Violations = append(result.Violations, bundleResult.Violations...)
		result.Warnings = append(result.Warnings, bundleResult.Warnings...)
	}

	return result, nil
}

func (e *Engine) evaluateRule(ctx context.Context, rule governance.PolicyRule, input *governance.PolicyInput) (*governance.PolicyResult, error) {
	result := &governance.PolicyResult{
		Allowed:    true,
		Violations: make([]governance.Violation, 0),
		Warnings:   make([]governance.Violation, 0),
	}

	// Build input map
	inputMap := map[string]any{
		"workflow_id":   input.WorkflowID,
		"workflow_name": input.WorkflowName,
		"run_id":        input.RunID,
		"step_name":     input.StepName,
		"agent_id":      input.AgentID,
		"agent_name":    input.AgentName,
		"capabilities":  input.Capabilities,
		"context":       input.Context,
		"metadata":      input.Metadata,
	}

	// Evaluate "allowed" query
	allowedQuery, err := rego.New(
		rego.Query("data.bridge.policy.allowed"),
		rego.Module("policy.rego", rule.Rego),
	).PrepareForEval(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to prepare allowed query: %w", err)
	}

	allowedResults, err := allowedQuery.Eval(ctx, rego.EvalInput(inputMap))
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate allowed query: %w", err)
	}

	if len(allowedResults) > 0 && len(allowedResults[0].Expressions) > 0 {
		if allowed, ok := allowedResults[0].Expressions[0].Value.(bool); ok {
			result.Allowed = allowed
		}
	}

	// Evaluate "requires_approval" query
	approvalQuery, err := rego.New(
		rego.Query("data.bridge.policy.requires_approval"),
		rego.Module("policy.rego", rule.Rego),
	).PrepareForEval(ctx)

	if err == nil {
		approvalResults, err := approvalQuery.Eval(ctx, rego.EvalInput(inputMap))
		if err == nil && len(approvalResults) > 0 && len(approvalResults[0].Expressions) > 0 {
			if requiresApproval, ok := approvalResults[0].Expressions[0].Value.(bool); ok {
				result.RequiresApproval = requiresApproval
			}
		}
	}

	// Evaluate "violation" query
	violationQuery, err := rego.New(
		rego.Query("data.bridge.policy.violation"),
		rego.Module("policy.rego", rule.Rego),
	).PrepareForEval(ctx)

	if err == nil {
		violationResults, err := violationQuery.Eval(ctx, rego.EvalInput(inputMap))
		if err == nil && len(violationResults) > 0 && len(violationResults[0].Expressions) > 0 {
			if violations, ok := violationResults[0].Expressions[0].Value.([]interface{}); ok {
				for _, v := range violations {
					msg := fmt.Sprintf("%v", v)
					violation := governance.Violation{
						Rule:     rule.Name,
						Message:  msg,
						Severity: rule.Severity,
					}

					if rule.Severity == governance.SeverityWarning || rule.Severity == governance.SeverityInfo {
						result.Warnings = append(result.Warnings, violation)
					} else {
						result.Violations = append(result.Violations, violation)
					}
				}
			}
		}
	}

	return result, nil
}

// ValidateRego validates a Rego policy.
func (e *Engine) ValidateRego(regoCode string) error {
	_, err := rego.New(
		rego.Query("data.bridge.policy"),
		rego.Module("validate.rego", regoCode),
	).PrepareForEval(context.Background())

	if err != nil {
		return fmt.Errorf("invalid rego policy: %w", err)
	}

	return nil
}

// Ensure Engine implements governance.Evaluator.
var _ governance.Evaluator = (*Engine)(nil)

// DefaultPolicies returns the Rego code for default policies.
func DefaultPolicies() string {
	return strings.TrimSpace(`
package bridge.policy

# Default allow
default allowed = true

# Default no approval required
default requires_approval = false

# Require approval for file write capabilities
requires_approval {
    input.capabilities[_] == "file-write"
}

# Require approval for shell execution
requires_approval {
    input.capabilities[_] == "shell-exec"
}

# Block access to sensitive paths
allowed = false {
    path := input.context.path
    contains(path, ".env")
}

allowed = false {
    path := input.context.path
    contains(path, "secrets/")
}

allowed = false {
    path := input.context.path
    contains(path, ".ssh/")
}

# Violation messages
violation[msg] {
    not allowed
    path := input.context.path
    msg := sprintf("Access to sensitive path blocked: %s", [path])
}

# Token limit enforcement
violation[msg] {
    input.context.max_tokens > 100000
    msg := sprintf("Token limit exceeded: %d > 100000", [input.context.max_tokens])
}
`)
}
