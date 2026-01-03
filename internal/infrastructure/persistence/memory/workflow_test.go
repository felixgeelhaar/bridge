package memory

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/bridge/internal/domain/workflow"
	"github.com/felixgeelhaar/bridge/pkg/config"
	"github.com/felixgeelhaar/bridge/pkg/types"
)

func createTestWorkflowDefinition(t *testing.T, name string) *workflow.WorkflowDefinition {
	t.Helper()
	cfg := &config.WorkflowConfig{
		Name:    name,
		Version: "1.0",
		Steps: []config.StepConfig{
			{Name: "step1", Agent: "test-agent"},
		},
	}
	def, err := workflow.NewWorkflowDefinition(cfg)
	if err != nil {
		t.Fatalf("Failed to create definition: %v", err)
	}
	return def
}

func TestNewWorkflowRepository(t *testing.T) {
	repo := NewWorkflowRepository()
	if repo == nil {
		t.Fatal("NewWorkflowRepository() returned nil")
	}
}

func TestWorkflowRepository_CreateDefinition(t *testing.T) {
	repo := NewWorkflowRepository()
	ctx := context.Background()

	def := createTestWorkflowDefinition(t, "test-workflow")
	err := repo.CreateDefinition(ctx, def)
	if err != nil {
		t.Fatalf("CreateDefinition() error = %v", err)
	}
}

func TestWorkflowRepository_CreateDefinition_Duplicate(t *testing.T) {
	repo := NewWorkflowRepository()
	ctx := context.Background()

	def1 := createTestWorkflowDefinition(t, "test-workflow")
	def2 := createTestWorkflowDefinition(t, "test-workflow")

	repo.CreateDefinition(ctx, def1)
	err := repo.CreateDefinition(ctx, def2)

	if err != types.ErrWorkflowAlreadyExists {
		t.Errorf("Expected ErrWorkflowAlreadyExists, got %v", err)
	}
}

func TestWorkflowRepository_GetDefinition(t *testing.T) {
	repo := NewWorkflowRepository()
	ctx := context.Background()

	def := createTestWorkflowDefinition(t, "test-workflow")
	repo.CreateDefinition(ctx, def)

	retrieved, err := repo.GetDefinition(ctx, def.ID)
	if err != nil {
		t.Fatalf("GetDefinition() error = %v", err)
	}

	if retrieved.ID != def.ID {
		t.Error("Retrieved definition ID should match")
	}
	if retrieved.Name != def.Name {
		t.Error("Retrieved definition name should match")
	}
}

func TestWorkflowRepository_GetDefinition_NotFound(t *testing.T) {
	repo := NewWorkflowRepository()
	ctx := context.Background()

	_, err := repo.GetDefinition(ctx, types.NewWorkflowID())
	if err != types.ErrWorkflowNotFound {
		t.Errorf("Expected ErrWorkflowNotFound, got %v", err)
	}
}

func TestWorkflowRepository_GetDefinitionByName(t *testing.T) {
	repo := NewWorkflowRepository()
	ctx := context.Background()

	def := createTestWorkflowDefinition(t, "named-workflow")
	repo.CreateDefinition(ctx, def)

	retrieved, err := repo.GetDefinitionByName(ctx, "named-workflow")
	if err != nil {
		t.Fatalf("GetDefinitionByName() error = %v", err)
	}

	if retrieved.Name != "named-workflow" {
		t.Error("Retrieved definition name should match")
	}
}

func TestWorkflowRepository_GetDefinitionByName_NotFound(t *testing.T) {
	repo := NewWorkflowRepository()
	ctx := context.Background()

	_, err := repo.GetDefinitionByName(ctx, "nonexistent")
	if err != types.ErrWorkflowNotFound {
		t.Errorf("Expected ErrWorkflowNotFound, got %v", err)
	}
}

func TestWorkflowRepository_ListDefinitions(t *testing.T) {
	repo := NewWorkflowRepository()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		def := createTestWorkflowDefinition(t, "workflow-"+string(rune('A'+i)))
		repo.CreateDefinition(ctx, def)
	}

	defs, err := repo.ListDefinitions(ctx, 10, 0)
	if err != nil {
		t.Fatalf("ListDefinitions() error = %v", err)
	}

	if len(defs) != 5 {
		t.Errorf("Expected 5 definitions, got %d", len(defs))
	}
}

func TestWorkflowRepository_ListDefinitions_Pagination(t *testing.T) {
	repo := NewWorkflowRepository()
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		def := createTestWorkflowDefinition(t, "workflow-"+string(rune('A'+i)))
		repo.CreateDefinition(ctx, def)
	}

	// First page
	page1, _ := repo.ListDefinitions(ctx, 3, 0)
	if len(page1) != 3 {
		t.Errorf("Page 1: expected 3, got %d", len(page1))
	}

	// Second page
	page2, _ := repo.ListDefinitions(ctx, 3, 3)
	if len(page2) != 3 {
		t.Errorf("Page 2: expected 3, got %d", len(page2))
	}

	// Offset beyond length
	empty, _ := repo.ListDefinitions(ctx, 10, 100)
	if len(empty) != 0 {
		t.Errorf("Offset beyond length: expected 0, got %d", len(empty))
	}
}

func TestWorkflowRepository_UpdateDefinition(t *testing.T) {
	repo := NewWorkflowRepository()
	ctx := context.Background()

	def := createTestWorkflowDefinition(t, "test-workflow")
	repo.CreateDefinition(ctx, def)

	def.Description = "Updated description"
	err := repo.UpdateDefinition(ctx, def)
	if err != nil {
		t.Fatalf("UpdateDefinition() error = %v", err)
	}

	retrieved, _ := repo.GetDefinition(ctx, def.ID)
	if retrieved.Description != "Updated description" {
		t.Error("Definition should be updated")
	}
}

func TestWorkflowRepository_UpdateDefinition_NotFound(t *testing.T) {
	repo := NewWorkflowRepository()
	ctx := context.Background()

	def := createTestWorkflowDefinition(t, "test-workflow")
	err := repo.UpdateDefinition(ctx, def)

	if err != types.ErrWorkflowNotFound {
		t.Errorf("Expected ErrWorkflowNotFound, got %v", err)
	}
}

func TestWorkflowRepository_DeleteDefinition(t *testing.T) {
	repo := NewWorkflowRepository()
	ctx := context.Background()

	def := createTestWorkflowDefinition(t, "test-workflow")
	repo.CreateDefinition(ctx, def)

	err := repo.DeleteDefinition(ctx, def.ID)
	if err != nil {
		t.Fatalf("DeleteDefinition() error = %v", err)
	}

	_, err = repo.GetDefinition(ctx, def.ID)
	if err != types.ErrWorkflowNotFound {
		t.Error("Definition should be deleted")
	}

	_, err = repo.GetDefinitionByName(ctx, "test-workflow")
	if err != types.ErrWorkflowNotFound {
		t.Error("Name index should be deleted")
	}
}

func TestWorkflowRepository_DeleteDefinition_NotFound(t *testing.T) {
	repo := NewWorkflowRepository()
	ctx := context.Background()

	err := repo.DeleteDefinition(ctx, types.NewWorkflowID())
	if err != types.ErrWorkflowNotFound {
		t.Errorf("Expected ErrWorkflowNotFound, got %v", err)
	}
}

func TestWorkflowRepository_CreateRun(t *testing.T) {
	repo := NewWorkflowRepository()
	ctx := context.Background()

	def := createTestWorkflowDefinition(t, "test-workflow")
	run := workflow.NewWorkflowRun(def, "test-trigger", nil)

	err := repo.CreateRun(ctx, run)
	if err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}
}

func TestWorkflowRepository_GetRun(t *testing.T) {
	repo := NewWorkflowRepository()
	ctx := context.Background()

	def := createTestWorkflowDefinition(t, "test-workflow")
	run := workflow.NewWorkflowRun(def, "test-trigger", nil)
	repo.CreateRun(ctx, run)

	retrieved, err := repo.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRun() error = %v", err)
	}

	if retrieved.ID != run.ID {
		t.Error("Retrieved run ID should match")
	}
}

func TestWorkflowRepository_GetRun_NotFound(t *testing.T) {
	repo := NewWorkflowRepository()
	ctx := context.Background()

	_, err := repo.GetRun(ctx, types.NewRunID())
	if err != types.ErrRunNotFound {
		t.Errorf("Expected ErrRunNotFound, got %v", err)
	}
}

func TestWorkflowRepository_ListRuns(t *testing.T) {
	repo := NewWorkflowRepository()
	ctx := context.Background()

	def := createTestWorkflowDefinition(t, "test-workflow")

	for i := 0; i < 3; i++ {
		run := workflow.NewWorkflowRun(def, "trigger", nil)
		repo.CreateRun(ctx, run)
	}

	runs, err := repo.ListRuns(ctx, def.ID, 10, 0)
	if err != nil {
		t.Fatalf("ListRuns() error = %v", err)
	}

	if len(runs) != 3 {
		t.Errorf("Expected 3 runs, got %d", len(runs))
	}
}

func TestWorkflowRepository_ListRuns_FiltersByWorkflow(t *testing.T) {
	repo := NewWorkflowRepository()
	ctx := context.Background()

	def1 := createTestWorkflowDefinition(t, "workflow-1")
	def2 := createTestWorkflowDefinition(t, "workflow-2")

	for i := 0; i < 2; i++ {
		repo.CreateRun(ctx, workflow.NewWorkflowRun(def1, "trigger", nil))
	}
	for i := 0; i < 3; i++ {
		repo.CreateRun(ctx, workflow.NewWorkflowRun(def2, "trigger", nil))
	}

	runs1, _ := repo.ListRuns(ctx, def1.ID, 10, 0)
	if len(runs1) != 2 {
		t.Errorf("Expected 2 runs for workflow-1, got %d", len(runs1))
	}

	runs2, _ := repo.ListRuns(ctx, def2.ID, 10, 0)
	if len(runs2) != 3 {
		t.Errorf("Expected 3 runs for workflow-2, got %d", len(runs2))
	}
}

func TestWorkflowRepository_ListActiveRuns(t *testing.T) {
	repo := NewWorkflowRepository()
	ctx := context.Background()

	def := createTestWorkflowDefinition(t, "test-workflow")

	run1 := workflow.NewWorkflowRun(def, "trigger", nil)
	run2 := workflow.NewWorkflowRun(def, "trigger", nil)
	run3 := workflow.NewWorkflowRun(def, "trigger", nil)

	run1.Start()
	run2.Start()
	run2.Complete() // Mark as completed
	run3.Start()

	repo.CreateRun(ctx, run1)
	repo.CreateRun(ctx, run2)
	repo.CreateRun(ctx, run3)

	activeRuns, err := repo.ListActiveRuns(ctx)
	if err != nil {
		t.Fatalf("ListActiveRuns() error = %v", err)
	}

	if len(activeRuns) != 2 {
		t.Errorf("Expected 2 active runs, got %d", len(activeRuns))
	}
}

func TestWorkflowRepository_UpdateRun(t *testing.T) {
	repo := NewWorkflowRepository()
	ctx := context.Background()

	def := createTestWorkflowDefinition(t, "test-workflow")
	run := workflow.NewWorkflowRun(def, "trigger", nil)
	repo.CreateRun(ctx, run)

	run.Start()
	err := repo.UpdateRun(ctx, run)
	if err != nil {
		t.Fatalf("UpdateRun() error = %v", err)
	}

	retrieved, _ := repo.GetRun(ctx, run.ID)
	if retrieved.Status != workflow.RunStatusPolicyCheck {
		t.Errorf("Run status should be policy_check, got %v", retrieved.Status)
	}
}

func TestWorkflowRepository_UpdateRun_NotFound(t *testing.T) {
	repo := NewWorkflowRepository()
	ctx := context.Background()

	def := createTestWorkflowDefinition(t, "test-workflow")
	run := workflow.NewWorkflowRun(def, "trigger", nil)

	err := repo.UpdateRun(ctx, run)
	if err != types.ErrRunNotFound {
		t.Errorf("Expected ErrRunNotFound, got %v", err)
	}
}

func TestWorkflowRepository_GetStep(t *testing.T) {
	repo := NewWorkflowRepository()
	ctx := context.Background()

	def := createTestWorkflowDefinition(t, "test-workflow")
	run := workflow.NewWorkflowRun(def, "trigger", nil)
	repo.CreateRun(ctx, run)

	step := run.Steps[0]
	retrieved, err := repo.GetStep(ctx, step.ID)
	if err != nil {
		t.Fatalf("GetStep() error = %v", err)
	}

	if retrieved.ID != step.ID {
		t.Error("Retrieved step ID should match")
	}
}

func TestWorkflowRepository_GetStep_NotFound(t *testing.T) {
	repo := NewWorkflowRepository()
	ctx := context.Background()

	_, err := repo.GetStep(ctx, types.NewStepID())
	if err != types.ErrStepNotFound {
		t.Errorf("Expected ErrStepNotFound, got %v", err)
	}
}

func TestWorkflowRepository_UpdateStep(t *testing.T) {
	repo := NewWorkflowRepository()
	ctx := context.Background()

	def := createTestWorkflowDefinition(t, "test-workflow")
	run := workflow.NewWorkflowRun(def, "trigger", nil)
	repo.CreateRun(ctx, run)

	step := run.Steps[0]
	step.Start(nil)
	err := repo.UpdateStep(ctx, step)
	if err != nil {
		t.Fatalf("UpdateStep() error = %v", err)
	}

	retrieved, _ := repo.GetStep(ctx, step.ID)
	if retrieved.Status != workflow.StepStatusRunning {
		t.Error("Step status should be updated")
	}
}

func TestWorkflowRepository_UpdateStep_NotFound(t *testing.T) {
	repo := NewWorkflowRepository()
	ctx := context.Background()

	step := &workflow.StepRun{ID: types.NewStepID()}
	err := repo.UpdateStep(ctx, step)
	if err != types.ErrStepNotFound {
		t.Errorf("Expected ErrStepNotFound, got %v", err)
	}
}

func TestWorkflowRepository_ImplementsRepository(t *testing.T) {
	var _ workflow.Repository = (*WorkflowRepository)(nil)
}
