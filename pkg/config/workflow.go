package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// WorkflowConfig represents a workflow definition loaded from YAML.
type WorkflowConfig struct {
	Name        string            `yaml:"name"`
	Version     string            `yaml:"version"`
	Description string            `yaml:"description,omitempty"`
	Triggers    []TriggerConfig   `yaml:"triggers,omitempty"`
	Steps       []StepConfig      `yaml:"steps"`
	Policies    []PolicyRefConfig `yaml:"policies,omitempty"`
	Metadata    map[string]any    `yaml:"metadata,omitempty"`
}

// TriggerConfig defines when a workflow should be triggered.
type TriggerConfig struct {
	Type   string   `yaml:"type"`
	Events []string `yaml:"events,omitempty"`
	Cron   string   `yaml:"cron,omitempty"`
	Filter string   `yaml:"filter,omitempty"`
}

// StepConfig defines a single step in a workflow.
type StepConfig struct {
	Name             string         `yaml:"name"`
	Agent            string         `yaml:"agent"`
	Input            map[string]any `yaml:"input,omitempty"`
	Output           string         `yaml:"output,omitempty"`
	RequiresApproval bool           `yaml:"requires_approval,omitempty"`
	Timeout          string         `yaml:"timeout,omitempty"`
	Retries          int            `yaml:"retries,omitempty"`
	Condition        string         `yaml:"condition,omitempty"`
	DependsOn        []string       `yaml:"depends_on,omitempty"`
}

// PolicyRefConfig references a policy to apply to the workflow.
type PolicyRefConfig struct {
	Name   string         `yaml:"name"`
	Rule   string         `yaml:"rule,omitempty"`
	Params map[string]any `yaml:"params,omitempty"`
}

// LoadWorkflow loads a workflow configuration from a YAML file.
func LoadWorkflow(path string) (*WorkflowConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow file: %w", err)
	}

	return ParseWorkflow(data)
}

// ParseWorkflow parses workflow configuration from YAML bytes.
func ParseWorkflow(data []byte) (*WorkflowConfig, error) {
	var config WorkflowConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse workflow YAML: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

// Validate validates the workflow configuration.
func (c *WorkflowConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("workflow name is required")
	}

	if c.Version == "" {
		return fmt.Errorf("workflow version is required")
	}

	if len(c.Steps) == 0 {
		return fmt.Errorf("workflow must have at least one step")
	}

	stepNames := make(map[string]bool)
	for i, step := range c.Steps {
		if step.Name == "" {
			return fmt.Errorf("step %d: name is required", i)
		}

		if stepNames[step.Name] {
			return fmt.Errorf("step %d: duplicate step name %q", i, step.Name)
		}
		stepNames[step.Name] = true

		if step.Agent == "" {
			return fmt.Errorf("step %q: agent is required", step.Name)
		}

		// Validate depends_on references
		for _, dep := range step.DependsOn {
			if !stepNames[dep] {
				return fmt.Errorf("step %q: depends_on references unknown step %q", step.Name, dep)
			}
		}
	}

	return nil
}

// ToYAML converts the workflow configuration to YAML bytes.
func (c *WorkflowConfig) ToYAML() ([]byte, error) {
	return yaml.Marshal(c)
}
