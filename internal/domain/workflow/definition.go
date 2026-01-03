package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/felixgeelhaar/bridge/pkg/config"
	"github.com/felixgeelhaar/bridge/pkg/types"
)

// WorkflowDefinition is the aggregate root for workflow definitions.
// It represents a reusable template for workflow execution.
type WorkflowDefinition struct {
	ID          types.WorkflowID
	Name        string
	Version     string
	Description string
	Steps       []StepDefinition
	Triggers    []Trigger
	Policies    []PolicyRef
	Checksum    string
	Metadata    map[string]any
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// StepDefinition defines a step template within a workflow.
type StepDefinition struct {
	Name             string
	AgentID          string
	Input            map[string]any
	Output           string
	RequiresApproval bool
	Timeout          time.Duration
	Retries          int
	Condition        string
	DependsOn        []string
}

// Trigger defines when a workflow should be executed.
type Trigger struct {
	Type   TriggerType
	Events []string
	Cron   string
	Filter string
}

// TriggerType represents the type of trigger.
type TriggerType string

const (
	TriggerTypeManual     TriggerType = "manual"
	TriggerTypeGitHubPR   TriggerType = "github.pull_request"
	TriggerTypeGitHubPush TriggerType = "github.push"
	TriggerTypeWebhook    TriggerType = "webhook"
	TriggerTypeCron       TriggerType = "cron"
)

// PolicyRef references a policy to be evaluated for this workflow.
type PolicyRef struct {
	Name   string
	Rule   string
	Params map[string]any
}

// NewWorkflowDefinition creates a new workflow definition from a config.
func NewWorkflowDefinition(cfg *config.WorkflowConfig) (*WorkflowDefinition, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	now := time.Now()
	def := &WorkflowDefinition{
		ID:          types.NewWorkflowID(),
		Name:        cfg.Name,
		Version:     cfg.Version,
		Description: cfg.Description,
		Steps:       make([]StepDefinition, 0, len(cfg.Steps)),
		Triggers:    make([]Trigger, 0, len(cfg.Triggers)),
		Policies:    make([]PolicyRef, 0, len(cfg.Policies)),
		Metadata:    cfg.Metadata,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Convert steps
	for _, s := range cfg.Steps {
		timeout := 5 * time.Minute // default
		if s.Timeout != "" {
			d, err := time.ParseDuration(s.Timeout)
			if err == nil {
				timeout = d
			}
		}

		def.Steps = append(def.Steps, StepDefinition{
			Name:             s.Name,
			AgentID:          s.Agent,
			Input:            s.Input,
			Output:           s.Output,
			RequiresApproval: s.RequiresApproval,
			Timeout:          timeout,
			Retries:          s.Retries,
			Condition:        s.Condition,
			DependsOn:        s.DependsOn,
		})
	}

	// Convert triggers
	for _, t := range cfg.Triggers {
		def.Triggers = append(def.Triggers, Trigger{
			Type:   TriggerType(t.Type),
			Events: t.Events,
			Cron:   t.Cron,
			Filter: t.Filter,
		})
	}

	// Convert policies
	for _, p := range cfg.Policies {
		def.Policies = append(def.Policies, PolicyRef{
			Name:   p.Name,
			Rule:   p.Rule,
			Params: p.Params,
		})
	}

	// Calculate checksum
	def.Checksum = def.calculateChecksum()

	return def, nil
}

// calculateChecksum generates a SHA256 checksum of the workflow definition.
func (d *WorkflowDefinition) calculateChecksum() string {
	data, _ := (&config.WorkflowConfig{
		Name:        d.Name,
		Version:     d.Version,
		Description: d.Description,
	}).ToYAML()
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// RequiresApproval returns true if any step requires approval.
func (d *WorkflowDefinition) RequiresApproval() bool {
	for _, s := range d.Steps {
		if s.RequiresApproval {
			return true
		}
	}
	return false
}

// GetStep returns a step by name.
func (d *WorkflowDefinition) GetStep(name string) *StepDefinition {
	for i := range d.Steps {
		if d.Steps[i].Name == name {
			return &d.Steps[i]
		}
	}
	return nil
}
