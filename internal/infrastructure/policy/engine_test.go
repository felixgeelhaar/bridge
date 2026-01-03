package policy_test

import (
	"context"
	"io"
	"testing"

	"github.com/felixgeelhaar/bolt"
	"github.com/felixgeelhaar/bridge/internal/domain/governance"
	"github.com/felixgeelhaar/bridge/internal/infrastructure/policy"
)

func newTestLogger() *bolt.Logger {
	handler := bolt.NewJSONHandler(io.Discard)
	return bolt.New(handler).SetLevel(bolt.ERROR) // Suppress logs during tests
}

func TestNewEngine(t *testing.T) {
	logger := newTestLogger()
	engine := policy.NewEngine(logger)

	if engine == nil {
		t.Fatal("expected non-nil engine")
	}
}

func TestEngine_LoadBundle(t *testing.T) {
	logger := newTestLogger()
	engine := policy.NewEngine(logger)

	bundle := governance.NewPolicyBundle("test", "1.0", "Test bundle")
	bundle.AddRule(governance.PolicyRule{
		Name:        "test_rule",
		Description: "A test rule",
		Enabled:     true,
		Severity:    governance.SeverityWarning,
		Rego: `
package bridge.policy
default allowed = true
`,
	})

	engine.LoadBundle(bundle)

	// Test passes if no panic occurs
}

func TestEngine_Evaluate_AllowedByDefault(t *testing.T) {
	logger := newTestLogger()
	engine := policy.NewEngine(logger)

	bundle := governance.NewPolicyBundle("test", "1.0", "Test bundle")
	bundle.AddRule(governance.PolicyRule{
		Name:        "allow_all",
		Description: "Allow everything",
		Enabled:     true,
		Severity:    governance.SeverityWarning,
		Rego: `
package bridge.policy
default allowed = true
`,
	})

	input := &governance.PolicyInput{
		WorkflowID:   "wf-123",
		WorkflowName: "test-workflow",
		RunID:        "run-456",
		StepName:     "step-1",
		Context:      map[string]any{"path": "/safe/path.txt"},
	}

	result, err := engine.Evaluate(context.Background(), bundle, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Allowed {
		t.Error("expected result to be allowed")
	}

	if len(result.Violations) > 0 {
		t.Errorf("expected no violations, got %d", len(result.Violations))
	}
}

func TestEngine_Evaluate_BlocksSensitivePaths(t *testing.T) {
	logger := newTestLogger()
	engine := policy.NewEngine(logger)

	bundle := governance.NewPolicyBundle("test", "1.0", "Test bundle")
	bundle.AddRule(governance.PolicyRule{
		Name:        "block_env",
		Description: "Block .env files",
		Enabled:     true,
		Severity:    governance.SeverityCritical,
		Rego: `
package bridge.policy

default allowed = true

allowed = false {
    contains(input.context.path, ".env")
}

violation[msg] {
    not allowed
    msg := "Access to .env file blocked"
}
`,
	})

	tests := []struct {
		name        string
		path        string
		wantAllowed bool
	}{
		{
			name:        "blocks .env file",
			path:        "/app/.env",
			wantAllowed: false,
		},
		{
			name:        "blocks .env.local",
			path:        "/app/.env.local",
			wantAllowed: false,
		},
		{
			name:        "allows regular file",
			path:        "/app/config.yaml",
			wantAllowed: true,
		},
		{
			name:        "allows src file",
			path:        "/app/src/main.go",
			wantAllowed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := &governance.PolicyInput{
				WorkflowID: "wf-123",
				RunID:      "run-456",
				Context:    map[string]any{"path": tt.path},
			}

			result, err := engine.Evaluate(context.Background(), bundle, input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Allowed != tt.wantAllowed {
				t.Errorf("allowed = %v, want %v", result.Allowed, tt.wantAllowed)
			}
		})
	}
}

func TestEngine_Evaluate_RequiresApproval(t *testing.T) {
	logger := newTestLogger()
	engine := policy.NewEngine(logger)

	bundle := governance.NewPolicyBundle("test", "1.0", "Test bundle")
	bundle.AddRule(governance.PolicyRule{
		Name:        "approval_for_writes",
		Description: "Require approval for write operations",
		Enabled:     true,
		Severity:    governance.SeverityWarning,
		Rego: `
package bridge.policy

default allowed = true
default requires_approval = false

requires_approval {
    input.capabilities[_] == "file-write"
}

requires_approval {
    input.capabilities[_] == "shell-exec"
}
`,
	})

	tests := []struct {
		name             string
		capabilities     []string
		wantApproval     bool
	}{
		{
			name:             "requires approval for file-write",
			capabilities:     []string{"file-read", "file-write"},
			wantApproval:     true,
		},
		{
			name:             "requires approval for shell-exec",
			capabilities:     []string{"shell-exec"},
			wantApproval:     true,
		},
		{
			name:             "no approval for read-only",
			capabilities:     []string{"file-read"},
			wantApproval:     false,
		},
		{
			name:             "no approval for empty capabilities",
			capabilities:     []string{},
			wantApproval:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := &governance.PolicyInput{
				WorkflowID:   "wf-123",
				RunID:        "run-456",
				Capabilities: tt.capabilities,
			}

			result, err := engine.Evaluate(context.Background(), bundle, input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.RequiresApproval != tt.wantApproval {
				t.Errorf("requires_approval = %v, want %v", result.RequiresApproval, tt.wantApproval)
			}
		})
	}
}

func TestEngine_Evaluate_DisabledRule(t *testing.T) {
	logger := newTestLogger()
	engine := policy.NewEngine(logger)

	bundle := governance.NewPolicyBundle("test", "1.0", "Test bundle")
	bundle.AddRule(governance.PolicyRule{
		Name:        "disabled_block",
		Description: "This rule is disabled",
		Enabled:     false, // Disabled
		Severity:    governance.SeverityCritical,
		Rego: `
package bridge.policy
default allowed = false
`,
	})

	input := &governance.PolicyInput{
		WorkflowID: "wf-123",
		RunID:      "run-456",
	}

	result, err := engine.Evaluate(context.Background(), bundle, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Disabled rules should not affect the result
	if !result.Allowed {
		t.Error("expected result to be allowed (disabled rule should be skipped)")
	}
}

func TestEngine_Evaluate_MultipleRules(t *testing.T) {
	logger := newTestLogger()
	engine := policy.NewEngine(logger)

	bundle := governance.NewPolicyBundle("test", "1.0", "Test bundle")

	// First rule: block secrets
	bundle.AddRule(governance.PolicyRule{
		Name:        "block_secrets",
		Description: "Block access to secrets",
		Enabled:     true,
		Severity:    governance.SeverityCritical,
		Rego: `
package bridge.policy

default allowed = true

allowed = false {
    contains(input.context.path, "secrets/")
}
`,
	})

	// Second rule: require approval for shell
	bundle.AddRule(governance.PolicyRule{
		Name:        "approval_shell",
		Description: "Require approval for shell",
		Enabled:     true,
		Severity:    governance.SeverityWarning,
		Rego: `
package bridge.policy

default requires_approval = false

requires_approval {
    input.capabilities[_] == "shell-exec"
}
`,
	})

	input := &governance.PolicyInput{
		WorkflowID:   "wf-123",
		RunID:        "run-456",
		Capabilities: []string{"shell-exec"},
		Context:      map[string]any{"path": "/secrets/api-key.txt"},
	}

	result, err := engine.Evaluate(context.Background(), bundle, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Allowed {
		t.Error("expected result to be blocked due to secrets path")
	}

	if !result.RequiresApproval {
		t.Error("expected result to require approval due to shell-exec")
	}
}

func TestEngine_EvaluateAll(t *testing.T) {
	logger := newTestLogger()
	engine := policy.NewEngine(logger)

	// Active bundle
	activeBundle := governance.NewPolicyBundle("active", "1.0", "Active bundle")
	activeBundle.Active = true
	activeBundle.AddRule(governance.PolicyRule{
		Name:        "require_approval",
		Description: "Require approval",
		Enabled:     true,
		Severity:    governance.SeverityWarning,
		Rego: `
package bridge.policy
default requires_approval = true
`,
	})

	// Inactive bundle
	inactiveBundle := governance.NewPolicyBundle("inactive", "1.0", "Inactive bundle")
	inactiveBundle.Active = false
	inactiveBundle.AddRule(governance.PolicyRule{
		Name:        "block_all",
		Description: "Block everything",
		Enabled:     true,
		Severity:    governance.SeverityCritical,
		Rego: `
package bridge.policy
default allowed = false
`,
	})

	engine.LoadBundle(activeBundle)
	engine.LoadBundle(inactiveBundle)

	input := &governance.PolicyInput{
		WorkflowID: "wf-123",
		RunID:      "run-456",
	}

	result, err := engine.EvaluateAll(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Active bundle should be evaluated
	if !result.RequiresApproval {
		t.Error("expected result to require approval from active bundle")
	}

	// Inactive bundle should be skipped, so allowed should still be true
	if !result.Allowed {
		t.Error("expected result to be allowed (inactive bundle should be skipped)")
	}
}

func TestEngine_Evaluate_Violations(t *testing.T) {
	logger := newTestLogger()
	engine := policy.NewEngine(logger)

	bundle := governance.NewPolicyBundle("test", "1.0", "Test bundle")
	bundle.AddRule(governance.PolicyRule{
		Name:        "token_limit",
		Description: "Enforce token limit",
		Enabled:     true,
		Severity:    governance.SeverityError,
		Rego: `
package bridge.policy

default allowed = true

allowed = false {
    input.context.max_tokens > 100000
}

violation[msg] {
    input.context.max_tokens > 100000
    msg := sprintf("Token limit exceeded: %d > 100000", [input.context.max_tokens])
}
`,
	})

	input := &governance.PolicyInput{
		WorkflowID: "wf-123",
		RunID:      "run-456",
		Context:    map[string]any{"max_tokens": 150000},
	}

	result, err := engine.Evaluate(context.Background(), bundle, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Allowed {
		t.Error("expected result to be blocked")
	}

	if len(result.Violations) == 0 {
		t.Error("expected at least one violation")
	}

	if len(result.Violations) > 0 {
		v := result.Violations[0]
		if v.Rule != "token_limit" {
			t.Errorf("violation rule = %s, want token_limit", v.Rule)
		}
		if v.Severity != governance.SeverityError {
			t.Errorf("violation severity = %s, want error", v.Severity)
		}
	}
}

func TestEngine_Evaluate_Warnings(t *testing.T) {
	logger := newTestLogger()
	engine := policy.NewEngine(logger)

	bundle := governance.NewPolicyBundle("test", "1.0", "Test bundle")
	bundle.AddRule(governance.PolicyRule{
		Name:        "large_file_warning",
		Description: "Warn about large files",
		Enabled:     true,
		Severity:    governance.SeverityWarning, // Warning severity
		Rego: `
package bridge.policy

default allowed = true

violation[msg] {
    input.context.file_size > 1000000
    msg := "Processing large file, may be slow"
}
`,
	})

	input := &governance.PolicyInput{
		WorkflowID: "wf-123",
		RunID:      "run-456",
		Context:    map[string]any{"file_size": 5000000},
	}

	result, err := engine.Evaluate(context.Background(), bundle, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should still be allowed (it's a warning)
	if !result.Allowed {
		t.Error("expected result to be allowed (warnings don't block)")
	}

	if len(result.Warnings) == 0 {
		t.Error("expected at least one warning")
	}

	if len(result.Violations) > 0 {
		t.Error("expected no violations (should be warnings)")
	}
}

func TestEngine_ValidateRego_Valid(t *testing.T) {
	logger := newTestLogger()
	engine := policy.NewEngine(logger)

	validRego := `
package bridge.policy

default allowed = true

allowed = false {
    input.context.blocked == true
}
`

	err := engine.ValidateRego(validRego)
	if err != nil {
		t.Errorf("unexpected error for valid rego: %v", err)
	}
}

func TestEngine_ValidateRego_Invalid(t *testing.T) {
	logger := newTestLogger()
	engine := policy.NewEngine(logger)

	invalidRego := `
package bridge.policy

default allowed = true

allowed = false {
    # Invalid syntax - missing closing brace
    input.context.blocked == true
`

	err := engine.ValidateRego(invalidRego)
	if err == nil {
		t.Error("expected error for invalid rego, got nil")
	}
}

func TestDefaultPolicies(t *testing.T) {
	policies := policy.DefaultPolicies()

	if policies == "" {
		t.Error("expected non-empty default policies")
	}

	// Validate the default policies are valid Rego
	logger := newTestLogger()
	engine := policy.NewEngine(logger)

	err := engine.ValidateRego(policies)
	if err != nil {
		t.Errorf("default policies are not valid rego: %v", err)
	}
}

func TestDefaultPolicyBundle(t *testing.T) {
	bundle := governance.DefaultPolicyBundle()

	if bundle == nil {
		t.Fatal("expected non-nil bundle")
	}

	if bundle.Name != "default" {
		t.Errorf("bundle name = %s, want default", bundle.Name)
	}

	if len(bundle.Rules) == 0 {
		t.Error("expected at least one rule in default bundle")
	}

	// Verify all rules have valid Rego
	logger := newTestLogger()
	engine := policy.NewEngine(logger)

	for _, rule := range bundle.Rules {
		err := engine.ValidateRego(rule.Rego)
		if err != nil {
			t.Errorf("rule %s has invalid rego: %v", rule.Name, err)
		}
	}
}

func TestPolicyResult_IsBlocking(t *testing.T) {
	tests := []struct {
		name    string
		result  governance.PolicyResult
		want    bool
	}{
		{
			name:    "not blocking when allowed",
			result:  governance.PolicyResult{Allowed: true},
			want:    false,
		},
		{
			name:    "blocking when not allowed",
			result:  governance.PolicyResult{Allowed: false},
			want:    true,
		},
		{
			name: "blocking on critical violation",
			result: governance.PolicyResult{
				Allowed: true,
				Violations: []governance.Violation{
					{Severity: governance.SeverityCritical},
				},
			},
			want: true,
		},
		{
			name: "blocking on error violation",
			result: governance.PolicyResult{
				Allowed: true,
				Violations: []governance.Violation{
					{Severity: governance.SeverityError},
				},
			},
			want: true,
		},
		{
			name: "not blocking on warning",
			result: governance.PolicyResult{
				Allowed: true,
				Violations: []governance.Violation{
					{Severity: governance.SeverityWarning},
				},
			},
			want: false,
		},
		{
			name: "not blocking on info",
			result: governance.PolicyResult{
				Allowed: true,
				Violations: []governance.Violation{
					{Severity: governance.SeverityInfo},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.result.IsBlocking()
			if got != tt.want {
				t.Errorf("IsBlocking() = %v, want %v", got, tt.want)
			}
		})
	}
}
