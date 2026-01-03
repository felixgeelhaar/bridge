package commands

import (
	"fmt"
	"os"

	"github.com/felixgeelhaar/bridge/internal/domain/workflow"
	"github.com/felixgeelhaar/bridge/internal/interfaces/cli/output"
	"github.com/felixgeelhaar/bridge/pkg/config"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

// ValidateCommand returns the validate command.
func ValidateCommand() *cli.Command {
	return &cli.Command{
		Name:  "validate",
		Usage: "Validate a workflow definition",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "workflow",
				Aliases:  []string{"w"},
				Usage:    "Path to workflow YAML file",
				Required: true,
			},
			&cli.BoolFlag{
				Name:  "strict",
				Usage: "Enable strict validation",
			},
		},
		Action: runValidate,
	}
}

func runValidate(c *cli.Context) error {
	formatter := output.NewFormatter(c.String("output"))
	workflowPath := c.String("workflow")
	strict := c.Bool("strict")

	// Read workflow file
	data, err := os.ReadFile(workflowPath)
	if err != nil {
		formatter.Error(fmt.Sprintf("Failed to read workflow file: %v", err))
		return err
	}

	// Parse YAML
	var cfg config.WorkflowConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		formatter.Error(fmt.Sprintf("Failed to parse workflow YAML: %v", err))
		return err
	}

	// Validate workflow
	errors, warnings := validateWorkflow(&cfg, strict)

	// Print results
	valid := len(errors) == 0
	formatter.ValidationResult(valid, errors, warnings)

	if !valid {
		return fmt.Errorf("validation failed")
	}

	// Try to create workflow definition to validate further
	_, err = workflow.NewWorkflowDefinition(&cfg)
	if err != nil {
		formatter.Error(fmt.Sprintf("Failed to create workflow definition: %v", err))
		return err
	}

	return nil
}

func validateWorkflow(cfg *config.WorkflowConfig, strict bool) ([]string, []string) {
	var errors []string
	var warnings []string

	// Required fields
	if cfg.Name == "" {
		errors = append(errors, "workflow name is required")
	}

	if cfg.Version == "" {
		errors = append(errors, "workflow version is required")
	}

	if len(cfg.Steps) == 0 {
		errors = append(errors, "at least one step is required")
	}

	// Validate steps
	stepNames := make(map[string]bool)
	for i, step := range cfg.Steps {
		if step.Name == "" {
			errors = append(errors, fmt.Sprintf("step %d: name is required", i+1))
		} else {
			if stepNames[step.Name] {
				errors = append(errors, fmt.Sprintf("step %d: duplicate step name '%s'", i+1, step.Name))
			}
			stepNames[step.Name] = true
		}

		if step.Agent == "" {
			errors = append(errors, fmt.Sprintf("step '%s': agent is required", step.Name))
		}

		// Validate step references in input
		if step.Input != nil {
			for key, value := range step.Input {
				if str, ok := value.(string); ok {
					refs := extractStepRefs(str)
					for _, ref := range refs {
						if !stepNames[ref] && ref != "trigger" {
							errors = append(errors, fmt.Sprintf("step '%s': input '%s' references unknown step '%s'", step.Name, key, ref))
						}
					}
				}
			}
		}
	}

	// Validate triggers
	if len(cfg.Triggers) == 0 {
		warnings = append(warnings, "no triggers defined - workflow can only be run manually")
	}

	for i, trigger := range cfg.Triggers {
		if trigger.Type == "" {
			errors = append(errors, fmt.Sprintf("trigger %d: type is required", i+1))
		}
	}

	// Validate policies
	for i, policy := range cfg.Policies {
		if policy.Name == "" {
			errors = append(errors, fmt.Sprintf("policy %d: name is required", i+1))
		}
		if policy.Rule == "" {
			warnings = append(warnings, fmt.Sprintf("policy '%s': rule is recommended", policy.Name))
		}
	}

	// Strict mode checks
	if strict {
		if cfg.Description == "" {
			warnings = append(warnings, "description is recommended")
		}

		for _, step := range cfg.Steps {
			if step.Timeout == "" {
				warnings = append(warnings, fmt.Sprintf("step '%s': timeout is recommended", step.Name))
			}
		}
	}

	return errors, warnings
}

// extractStepRefs extracts step references from a string like "${{ steps.name.output }}"
func extractStepRefs(s string) []string {
	var refs []string
	// Simple extraction - look for ${{ steps.NAME.
	start := 0
	for {
		idx := indexOf(s[start:], "${{ steps.")
		if idx == -1 {
			break
		}
		idx += start + len("${{ steps.")
		end := idx
		for end < len(s) && s[end] != '.' && s[end] != ' ' && s[end] != '}' {
			end++
		}
		if end > idx {
			refs = append(refs, s[idx:end])
		}
		start = end
	}
	return refs
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
