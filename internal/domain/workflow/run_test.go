package workflow

import (
	"testing"

	"github.com/felixgeelhaar/bridge/pkg/config"
)

func createTestDefinition(t *testing.T) *WorkflowDefinition {
	t.Helper()
	cfg := &config.WorkflowConfig{
		Name:    "test-workflow",
		Version: "1.0",
		Steps: []config.StepConfig{
			{Name: "step1", Agent: "agent1"},
			{Name: "step2", Agent: "agent2"},
			{Name: "step3", Agent: "agent3"},
		},
	}

	def, err := NewWorkflowDefinition(cfg)
	if err != nil {
		t.Fatalf("Failed to create definition: %v", err)
	}
	return def
}

func TestNewWorkflowRun(t *testing.T) {
	def := createTestDefinition(t)
	triggerData := map[string]any{
		"pr_number": 123,
		"repo":      "owner/repo",
	}

	run := NewWorkflowRun(def, "test-trigger", triggerData)

	if run == nil {
		t.Fatal("NewWorkflowRun returned nil")
	}

	if run.WorkflowID != def.ID {
		t.Errorf("WorkflowID = %v, want %v", run.WorkflowID, def.ID)
	}

	if run.WorkflowName != def.Name {
		t.Errorf("WorkflowName = %v, want %v", run.WorkflowName, def.Name)
	}

	if run.Status != RunStatusPending {
		t.Errorf("Status = %v, want %v", run.Status, RunStatusPending)
	}

	if run.TriggeredBy != "test-trigger" {
		t.Errorf("TriggeredBy = %v, want test-trigger", run.TriggeredBy)
	}

	if len(run.Steps) != len(def.Steps) {
		t.Errorf("Steps count = %v, want %v", len(run.Steps), len(def.Steps))
	}

	// Verify trigger data is stored
	if run.TriggerData["pr_number"] != 123 {
		t.Error("Trigger data not properly stored")
	}
}

func TestWorkflowRun_Start(t *testing.T) {
	def := createTestDefinition(t)
	run := NewWorkflowRun(def, "test", nil)

	run.Start()

	// Start transitions to policy_check first (before executing)
	if run.Status != RunStatusPolicyCheck {
		t.Errorf("Status after Start = %v, want %v", run.Status, RunStatusPolicyCheck)
	}

	if run.StartedAt == nil {
		t.Error("StartedAt should be set after Start")
	}
}

func TestWorkflowRun_Execute(t *testing.T) {
	def := createTestDefinition(t)
	run := NewWorkflowRun(def, "test", nil)

	run.Start()
	run.Execute()

	if run.Status != RunStatusExecuting {
		t.Errorf("Status after Execute = %v, want %v", run.Status, RunStatusExecuting)
	}
}

func TestWorkflowRun_Complete(t *testing.T) {
	def := createTestDefinition(t)
	run := NewWorkflowRun(def, "test", nil)

	run.Start()
	run.Complete()

	if run.Status != RunStatusCompleted {
		t.Errorf("Status after Complete = %v, want %v", run.Status, RunStatusCompleted)
	}

	if run.CompletedAt == nil {
		t.Error("CompletedAt should be set after Complete")
	}
}

func TestWorkflowRun_Fail(t *testing.T) {
	def := createTestDefinition(t)
	run := NewWorkflowRun(def, "test", nil)

	run.Start()
	run.Fail("test error")

	if run.Status != RunStatusFailed {
		t.Errorf("Status after Fail = %v, want %v", run.Status, RunStatusFailed)
	}

	if run.Error != "test error" {
		t.Errorf("Error = %v, want 'test error'", run.Error)
	}
}

func TestWorkflowRun_AwaitApproval(t *testing.T) {
	def := createTestDefinition(t)
	run := NewWorkflowRun(def, "test", nil)

	run.Start()
	run.AwaitApproval()

	if run.Status != RunStatusAwaitingApproval {
		t.Errorf("Status = %v, want %v", run.Status, RunStatusAwaitingApproval)
	}
}

func TestWorkflowRun_Approve(t *testing.T) {
	def := createTestDefinition(t)
	run := NewWorkflowRun(def, "test", nil)

	run.Start()
	run.AwaitApproval()
	run.Approve()

	if run.Status != RunStatusExecuting {
		t.Errorf("Status after Approve = %v, want %v", run.Status, RunStatusExecuting)
	}
}

func TestWorkflowRun_StepNavigation(t *testing.T) {
	def := createTestDefinition(t)
	run := NewWorkflowRun(def, "test", nil)

	// Initially should have more steps
	if !run.HasMoreSteps() {
		t.Error("HasMoreSteps should return true initially")
	}

	// Get first step
	step := run.CurrentStep()
	if step == nil {
		t.Fatal("CurrentStep returned nil")
	}
	if step.Name != "step1" {
		t.Errorf("First step name = %v, want step1", step.Name)
	}

	// Advance to next step
	run.AdvanceStep()
	step = run.CurrentStep()
	if step.Name != "step2" {
		t.Errorf("Second step name = %v, want step2", step.Name)
	}

	// Advance again
	run.AdvanceStep()
	step = run.CurrentStep()
	if step.Name != "step3" {
		t.Errorf("Third step name = %v, want step3", step.Name)
	}

	// Advance past last step
	run.AdvanceStep()
	if run.HasMoreSteps() {
		t.Error("HasMoreSteps should return false after all steps")
	}
}

func TestWorkflowRun_SetContext(t *testing.T) {
	def := createTestDefinition(t)
	run := NewWorkflowRun(def, "test", nil)

	run.SetContext("key1", "value1")
	run.SetContext("key2", 42)
	run.SetContext("key3", map[string]string{"nested": "value"})

	if run.Context["key1"] != "value1" {
		t.Error("SetContext failed for string value")
	}
	if run.Context["key2"] != 42 {
		t.Error("SetContext failed for int value")
	}
	if run.Context["key3"] == nil {
		t.Error("SetContext failed for map value")
	}
}

func TestWorkflowRun_Duration(t *testing.T) {
	def := createTestDefinition(t)
	run := NewWorkflowRun(def, "test", nil)

	// Duration before start should be 0
	if run.Duration() != 0 {
		t.Error("Duration should be 0 before start")
	}

	run.Start()
	run.Complete()

	// Duration after completion should be positive
	if run.Duration() <= 0 {
		t.Error("Duration should be positive after completion")
	}
}
