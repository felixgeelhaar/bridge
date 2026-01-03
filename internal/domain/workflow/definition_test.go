package workflow

import (
	"testing"

	"github.com/felixgeelhaar/bridge/pkg/config"
)

func TestNewWorkflowDefinition(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.WorkflowConfig
		wantErr bool
	}{
		{
			name: "valid workflow",
			cfg: &config.WorkflowConfig{
				Name:        "test-workflow",
				Version:     "1.0",
				Description: "Test workflow",
				Steps: []config.StepConfig{
					{
						Name:  "step1",
						Agent: "code-reviewer",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "workflow with multiple steps",
			cfg: &config.WorkflowConfig{
				Name:    "multi-step",
				Version: "1.0",
				Steps: []config.StepConfig{
					{Name: "step1", Agent: "agent1"},
					{Name: "step2", Agent: "agent2"},
					{Name: "step3", Agent: "agent3"},
				},
			},
			wantErr: false,
		},
		{
			name: "workflow with triggers",
			cfg: &config.WorkflowConfig{
				Name:    "triggered",
				Version: "1.0",
				Triggers: []config.TriggerConfig{
					{Type: "github.pull_request", Events: []string{"opened", "synchronize"}},
				},
				Steps: []config.StepConfig{
					{Name: "step1", Agent: "agent1"},
				},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			cfg: &config.WorkflowConfig{
				Version: "1.0",
				Steps: []config.StepConfig{
					{Name: "step1", Agent: "agent1"},
				},
			},
			wantErr: true,
		},
		{
			name: "missing version",
			cfg: &config.WorkflowConfig{
				Name: "test",
				Steps: []config.StepConfig{
					{Name: "step1", Agent: "agent1"},
				},
			},
			wantErr: true,
		},
		{
			name: "no steps",
			cfg: &config.WorkflowConfig{
				Name:    "test",
				Version: "1.0",
				Steps:   []config.StepConfig{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def, err := NewWorkflowDefinition(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewWorkflowDefinition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && def == nil {
				t.Error("NewWorkflowDefinition() returned nil definition")
			}
			if !tt.wantErr {
				if def.Name != tt.cfg.Name {
					t.Errorf("Name = %v, want %v", def.Name, tt.cfg.Name)
				}
				if def.Version != tt.cfg.Version {
					t.Errorf("Version = %v, want %v", def.Version, tt.cfg.Version)
				}
				if len(def.Steps) != len(tt.cfg.Steps) {
					t.Errorf("Steps count = %v, want %v", len(def.Steps), len(tt.cfg.Steps))
				}
			}
		})
	}
}

func TestWorkflowDefinition_GetStep(t *testing.T) {
	cfg := &config.WorkflowConfig{
		Name:    "test",
		Version: "1.0",
		Steps: []config.StepConfig{
			{Name: "step1", Agent: "agent1"},
			{Name: "step2", Agent: "agent2"},
		},
	}

	def, err := NewWorkflowDefinition(cfg)
	if err != nil {
		t.Fatalf("Failed to create definition: %v", err)
	}

	// Test getting existing step
	step := def.GetStep("step1")
	if step == nil {
		t.Error("GetStep() returned nil for existing step")
	}
	if step != nil && step.Name != "step1" {
		t.Errorf("GetStep() returned wrong step: %v", step.Name)
	}

	// Test getting non-existent step
	step = def.GetStep("nonexistent")
	if step != nil {
		t.Error("GetStep() should return nil for non-existent step")
	}
}
