package orchestrator

import (
	"context"
	"os"
	"testing"

	"github.com/felixgeelhaar/bolt"
	"github.com/felixgeelhaar/bridge/internal/domain/agents"
	"github.com/felixgeelhaar/bridge/internal/domain/governance"
	"github.com/felixgeelhaar/bridge/internal/domain/workflow"
	"github.com/felixgeelhaar/bridge/internal/infrastructure/llm"
	"github.com/felixgeelhaar/bridge/internal/infrastructure/messaging/eventbus"
	"github.com/felixgeelhaar/bridge/internal/infrastructure/persistence/memory"
	"github.com/felixgeelhaar/bridge/internal/infrastructure/policy"
	"github.com/felixgeelhaar/bridge/pkg/config"
)

func createTestOrchestrator(t *testing.T) *Orchestrator {
	t.Helper()

	handler := bolt.NewConsoleHandler(os.Stderr)
	logger := bolt.New(handler).SetLevel(bolt.ERROR)

	llmRegistry := llm.NewRegistry()
	agentRegistry := agents.NewAgentRegistry()
	workflowRepo := memory.NewWorkflowRepository()
	eventPublisher := eventbus.New()
	policyEngine := policy.NewEngine(logger)
	auditLogger := governance.NewInMemoryAuditLogger()

	// Load default policy
	defaultBundle := &governance.PolicyBundle{
		Name:    "default",
		Version: "1.0",
		Active:  true,
		Rules: []governance.PolicyRule{
			{
				Name:     "allow-all",
				Enabled:  true,
				Rego:     policy.DefaultPolicies(),
				Severity: governance.SeverityInfo,
			},
		},
	}
	policyEngine.LoadBundle(defaultBundle)

	orch, err := New(Config{
		Logger:          logger,
		WorkflowRepo:    workflowRepo,
		EventPublisher:  eventPublisher,
		PolicyEvaluator: policyEngine,
		AuditLogger:     auditLogger,
		LLMRegistry:     llmRegistry,
		AgentRegistry:   agentRegistry,
	})
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}

	return orch
}

func TestNew(t *testing.T) {
	handler := bolt.NewConsoleHandler(os.Stderr)
	logger := bolt.New(handler).SetLevel(bolt.ERROR)

	cfg := Config{
		Logger:          logger,
		WorkflowRepo:    memory.NewWorkflowRepository(),
		EventPublisher:  eventbus.New(),
		PolicyEvaluator: policy.NewEngine(logger),
		AuditLogger:     governance.NewInMemoryAuditLogger(),
		LLMRegistry:     llm.NewRegistry(),
		AgentRegistry:   agents.NewAgentRegistry(),
	}

	orch, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if orch == nil {
		t.Fatal("New() returned nil")
	}
}

func TestOrchestrator_CreateWorkflow(t *testing.T) {
	orch := createTestOrchestrator(t)
	ctx := context.Background()

	cfg := &config.WorkflowConfig{
		Name:        "test-workflow",
		Version:     "1.0",
		Description: "Test workflow",
		Steps: []config.StepConfig{
			{Name: "step1", Agent: "test-agent"},
		},
	}

	def, err := orch.CreateWorkflow(ctx, cfg)
	if err != nil {
		t.Fatalf("CreateWorkflow() error = %v", err)
	}

	if def == nil {
		t.Fatal("CreateWorkflow() returned nil")
	}
	if def.Name != cfg.Name {
		t.Errorf("Workflow name = %v, want %v", def.Name, cfg.Name)
	}
	if def.Version != cfg.Version {
		t.Errorf("Workflow version = %v, want %v", def.Version, cfg.Version)
	}
}

func TestOrchestrator_CreateRun(t *testing.T) {
	orch := createTestOrchestrator(t)
	ctx := context.Background()

	// First create a workflow
	cfg := &config.WorkflowConfig{
		Name:    "test-workflow",
		Version: "1.0",
		Steps: []config.StepConfig{
			{Name: "step1", Agent: "test-agent"},
		},
	}

	def, err := orch.CreateWorkflow(ctx, cfg)
	if err != nil {
		t.Fatalf("CreateWorkflow() error = %v", err)
	}

	// Create a run
	triggerData := map[string]any{
		"key": "value",
	}
	run, err := orch.CreateRun(ctx, def, "test-trigger", triggerData)
	if err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}

	if run == nil {
		t.Fatal("CreateRun() returned nil")
	}
	if run.WorkflowID != def.ID {
		t.Errorf("Run WorkflowID = %v, want %v", run.WorkflowID, def.ID)
	}
	if run.TriggeredBy != "test-trigger" {
		t.Errorf("Run TriggeredBy = %v, want test-trigger", run.TriggeredBy)
	}
	if run.Status != workflow.RunStatusPending {
		t.Errorf("Run Status = %v, want %v", run.Status, workflow.RunStatusPending)
	}
}

func TestOrchestrator_GetRun(t *testing.T) {
	orch := createTestOrchestrator(t)
	ctx := context.Background()

	// Create workflow and run
	cfg := &config.WorkflowConfig{
		Name:    "test-workflow",
		Version: "1.0",
		Steps: []config.StepConfig{
			{Name: "step1", Agent: "test-agent"},
		},
	}

	def, _ := orch.CreateWorkflow(ctx, cfg)
	run, _ := orch.CreateRun(ctx, def, "test", nil)

	// Get the run
	retrieved, err := orch.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRun() error = %v", err)
	}

	if retrieved.ID != run.ID {
		t.Errorf("Retrieved run ID = %v, want %v", retrieved.ID, run.ID)
	}
}

func TestOrchestrator_ListActiveRuns(t *testing.T) {
	orch := createTestOrchestrator(t)
	ctx := context.Background()

	// Initially no active runs
	runs, err := orch.ListActiveRuns(ctx)
	if err != nil {
		t.Fatalf("ListActiveRuns() error = %v", err)
	}
	if len(runs) != 0 {
		t.Errorf("Initially should have 0 active runs, got %d", len(runs))
	}

	// Create workflow and runs
	cfg := &config.WorkflowConfig{
		Name:    "test-workflow",
		Version: "1.0",
		Steps: []config.StepConfig{
			{Name: "step1", Agent: "test-agent"},
		},
	}

	def, _ := orch.CreateWorkflow(ctx, cfg)
	orch.CreateRun(ctx, def, "test1", nil)
	orch.CreateRun(ctx, def, "test2", nil)

	// Should have 2 active runs
	runs, err = orch.ListActiveRuns(ctx)
	if err != nil {
		t.Fatalf("ListActiveRuns() error = %v", err)
	}
	if len(runs) != 2 {
		t.Errorf("Should have 2 active runs, got %d", len(runs))
	}
}

func TestConfig_Defaults(t *testing.T) {
	// Test that Config can be created with minimal setup
	handler := bolt.NewConsoleHandler(os.Stderr)
	logger := bolt.New(handler)

	cfg := Config{
		Logger:          logger,
		WorkflowRepo:    memory.NewWorkflowRepository(),
		EventPublisher:  eventbus.New(),
		PolicyEvaluator: policy.NewEngine(logger),
		AuditLogger:     governance.NewInMemoryAuditLogger(),
		LLMRegistry:     llm.NewRegistry(),
		AgentRegistry:   agents.NewAgentRegistry(),
	}

	if cfg.Logger == nil {
		t.Error("Logger should not be nil")
	}
	if cfg.WorkflowRepo == nil {
		t.Error("WorkflowRepo should not be nil")
	}
}
