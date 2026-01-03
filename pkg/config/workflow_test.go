package config

import (
	"testing"
)

func TestParseWorkflow(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
		check   func(*testing.T, *WorkflowConfig)
	}{
		{
			name: "simple workflow",
			yaml: `
name: test-workflow
version: "1.0"
steps:
  - name: step1
    agent: code-reviewer
`,
			wantErr: false,
			check: func(t *testing.T, cfg *WorkflowConfig) {
				if cfg.Name != "test-workflow" {
					t.Errorf("Name = %v, want test-workflow", cfg.Name)
				}
				if cfg.Version != "1.0" {
					t.Errorf("Version = %v, want 1.0", cfg.Version)
				}
				if len(cfg.Steps) != 1 {
					t.Errorf("Steps count = %v, want 1", len(cfg.Steps))
				}
			},
		},
		{
			name: "workflow with description",
			yaml: `
name: documented
version: "2.0"
description: This is a well-documented workflow
steps:
  - name: step1
    agent: agent1
`,
			wantErr: false,
			check: func(t *testing.T, cfg *WorkflowConfig) {
				if cfg.Description != "This is a well-documented workflow" {
					t.Errorf("Description not parsed correctly")
				}
			},
		},
		{
			name: "workflow with triggers",
			yaml: `
name: triggered
version: "1.0"
triggers:
  - type: github.pull_request
    events:
      - opened
      - synchronize
  - type: cron
    cron: "0 0 * * *"
steps:
  - name: step1
    agent: agent1
`,
			wantErr: false,
			check: func(t *testing.T, cfg *WorkflowConfig) {
				if len(cfg.Triggers) != 2 {
					t.Errorf("Triggers count = %v, want 2", len(cfg.Triggers))
				}
				if cfg.Triggers[0].Type != "github.pull_request" {
					t.Errorf("First trigger type = %v", cfg.Triggers[0].Type)
				}
				if len(cfg.Triggers[0].Events) != 2 {
					t.Errorf("Events count = %v, want 2", len(cfg.Triggers[0].Events))
				}
				if cfg.Triggers[1].Cron != "0 0 * * *" {
					t.Errorf("Cron = %v", cfg.Triggers[1].Cron)
				}
			},
		},
		{
			name: "workflow with step options",
			yaml: `
name: options
version: "1.0"
steps:
  - name: step1
    agent: agent1
    timeout: "5m"
    retries: 3
    requires_approval: true
    condition: "${{ steps.previous.output.success }}"
`,
			wantErr: false,
			check: func(t *testing.T, cfg *WorkflowConfig) {
				step := cfg.Steps[0]
				if step.Timeout != "5m" {
					t.Errorf("Timeout = %v, want 5m", step.Timeout)
				}
				if step.Retries != 3 {
					t.Errorf("Retries = %v, want 3", step.Retries)
				}
				if !step.RequiresApproval {
					t.Error("RequiresApproval should be true")
				}
				if step.Condition == "" {
					t.Error("Condition should be set")
				}
			},
		},
		{
			name: "workflow with step inputs",
			yaml: `
name: inputs
version: "1.0"
steps:
  - name: step1
    agent: agent1
    input:
      key1: value1
      key2: 42
      key3:
        nested: value
`,
			wantErr: false,
			check: func(t *testing.T, cfg *WorkflowConfig) {
				if cfg.Steps[0].Input == nil {
					t.Fatal("Input should not be nil")
				}
				if cfg.Steps[0].Input["key1"] != "value1" {
					t.Error("Input key1 not parsed correctly")
				}
				if cfg.Steps[0].Input["key2"] != 42 {
					t.Error("Input key2 not parsed correctly")
				}
			},
		},
		{
			name: "workflow with dependencies",
			yaml: `
name: deps
version: "1.0"
steps:
  - name: step1
    agent: agent1
  - name: step2
    agent: agent2
    depends_on:
      - step1
  - name: step3
    agent: agent3
    depends_on:
      - step1
      - step2
`,
			wantErr: false,
			check: func(t *testing.T, cfg *WorkflowConfig) {
				if len(cfg.Steps[1].DependsOn) != 1 {
					t.Error("step2 should have 1 dependency")
				}
				if len(cfg.Steps[2].DependsOn) != 2 {
					t.Error("step3 should have 2 dependencies")
				}
			},
		},
		{
			name: "workflow with policies",
			yaml: `
name: governed
version: "1.0"
steps:
  - name: step1
    agent: agent1
policies:
  - name: require-approval
    rule: "steps.*.requires_approval"
  - name: rate-limit
    rule: "ratelimit(user, '10/hour')"
    params:
      scope: user
`,
			wantErr: false,
			check: func(t *testing.T, cfg *WorkflowConfig) {
				if len(cfg.Policies) != 2 {
					t.Errorf("Policies count = %v, want 2", len(cfg.Policies))
				}
				if cfg.Policies[0].Name != "require-approval" {
					t.Error("First policy name incorrect")
				}
				if cfg.Policies[1].Params == nil {
					t.Error("Second policy should have params")
				}
			},
		},
		{
			name: "workflow with metadata",
			yaml: `
name: metadata
version: "1.0"
steps:
  - name: step1
    agent: agent1
metadata:
  owner: team-name
  environment: production
`,
			wantErr: false,
			check: func(t *testing.T, cfg *WorkflowConfig) {
				if cfg.Metadata == nil {
					t.Fatal("Metadata should not be nil")
				}
				if cfg.Metadata["owner"] != "team-name" {
					t.Error("Metadata owner incorrect")
				}
			},
		},
		{
			name:    "missing name",
			yaml:    `version: "1.0"`,
			wantErr: true,
		},
		{
			name:    "missing version",
			yaml:    `name: test`,
			wantErr: true,
		},
		{
			name: "missing steps",
			yaml: `
name: test
version: "1.0"
`,
			wantErr: true,
		},
		{
			name:    "invalid yaml",
			yaml:    `{not: valid: yaml`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := ParseWorkflow([]byte(tt.yaml))
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseWorkflow() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}

func TestWorkflowConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     WorkflowConfig
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: WorkflowConfig{
				Name:    "test",
				Version: "1.0",
				Steps: []StepConfig{
					{Name: "step1", Agent: "agent1"},
				},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			cfg: WorkflowConfig{
				Version: "1.0",
				Steps: []StepConfig{
					{Name: "step1", Agent: "agent1"},
				},
			},
			wantErr: true,
		},
		{
			name: "missing version",
			cfg: WorkflowConfig{
				Name: "test",
				Steps: []StepConfig{
					{Name: "step1", Agent: "agent1"},
				},
			},
			wantErr: true,
		},
		{
			name: "empty steps",
			cfg: WorkflowConfig{
				Name:    "test",
				Version: "1.0",
				Steps:   []StepConfig{},
			},
			wantErr: true,
		},
		{
			name: "step without name",
			cfg: WorkflowConfig{
				Name:    "test",
				Version: "1.0",
				Steps: []StepConfig{
					{Agent: "agent1"},
				},
			},
			wantErr: true,
		},
		{
			name: "step without agent",
			cfg: WorkflowConfig{
				Name:    "test",
				Version: "1.0",
				Steps: []StepConfig{
					{Name: "step1"},
				},
			},
			wantErr: true,
		},
		{
			name: "duplicate step names",
			cfg: WorkflowConfig{
				Name:    "test",
				Version: "1.0",
				Steps: []StepConfig{
					{Name: "step1", Agent: "agent1"},
					{Name: "step1", Agent: "agent2"},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid depends_on reference",
			cfg: WorkflowConfig{
				Name:    "test",
				Version: "1.0",
				Steps: []StepConfig{
					{Name: "step1", Agent: "agent1", DependsOn: []string{"nonexistent"}},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWorkflowConfig_ToYAML(t *testing.T) {
	cfg := &WorkflowConfig{
		Name:        "test",
		Version:     "1.0",
		Description: "Test workflow",
		Steps: []StepConfig{
			{Name: "step1", Agent: "agent1"},
		},
	}

	data, err := cfg.ToYAML()
	if err != nil {
		t.Fatalf("ToYAML() error = %v", err)
	}

	// Parse it back
	parsed, err := ParseWorkflow(data)
	if err != nil {
		t.Fatalf("Failed to parse YAML output: %v", err)
	}

	if parsed.Name != cfg.Name {
		t.Errorf("Round-trip Name = %v, want %v", parsed.Name, cfg.Name)
	}
	if parsed.Version != cfg.Version {
		t.Errorf("Round-trip Version = %v, want %v", parsed.Version, cfg.Version)
	}
}
