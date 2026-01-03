package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/felixgeelhaar/bolt"
	"github.com/felixgeelhaar/bridge/internal/application/orchestrator"
	"github.com/felixgeelhaar/bridge/internal/domain/agents"
	"github.com/felixgeelhaar/bridge/internal/domain/governance"
	"github.com/felixgeelhaar/bridge/internal/infrastructure/llm"
	"github.com/felixgeelhaar/bridge/internal/infrastructure/messaging/eventbus"
	"github.com/felixgeelhaar/bridge/internal/infrastructure/persistence/memory"
	"github.com/felixgeelhaar/bridge/internal/infrastructure/policy"
	"github.com/felixgeelhaar/bridge/internal/interfaces/cli/output"
	"github.com/felixgeelhaar/bridge/pkg/types"
	"github.com/google/uuid"
	"github.com/urfave/cli/v2"
)

// StatusCommand returns the status command.
func StatusCommand() *cli.Command {
	return &cli.Command{
		Name:      "status",
		Usage:     "Check status of a workflow run",
		ArgsUsage: "[run-id]",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
				Usage:   "List all active runs",
			},
			&cli.BoolFlag{
				Name:  "watch",
				Usage: "Watch for status changes",
			},
		},
		Action: runStatus,
	}
}

func runStatus(c *cli.Context) error {
	formatter := output.NewFormatter(c.String("output"))
	listAll := c.Bool("all")

	// Setup minimal infrastructure for status checks
	ctx := context.Background()
	handler := bolt.NewConsoleHandler(os.Stderr)
	logger := bolt.New(handler).SetLevel(bolt.ERROR) // Quiet for status

	// Create orchestrator
	orch, err := createMinimalOrchestrator(logger)
	if err != nil {
		formatter.Error(fmt.Sprintf("Failed to initialize: %v", err))
		return err
	}

	if listAll {
		// List all active runs
		runs, err := orch.ListActiveRuns(ctx)
		if err != nil {
			formatter.Error(fmt.Sprintf("Failed to list runs: %v", err))
			return err
		}

		formatter.RunList(runs)
		return nil
	}

	// Get specific run
	runID := c.Args().First()
	if runID == "" {
		formatter.Error("Run ID required. Use --all to list all runs.")
		return fmt.Errorf("run id required")
	}

	// Parse run ID
	id, err := uuid.Parse(runID)
	if err != nil {
		// Try partial match
		formatter.Error(fmt.Sprintf("Invalid run ID: %s", runID))
		return err
	}

	run, err := orch.GetRun(ctx, types.RunID(id.String()))
	if err != nil {
		formatter.Error(fmt.Sprintf("Run not found: %s", runID))
		return err
	}

	formatter.WorkflowRun(run)

	return nil
}

func createMinimalOrchestrator(logger *bolt.Logger) (*orchestrator.Orchestrator, error) {
	llmRegistry := llm.NewRegistry()
	agentRegistry := agents.NewAgentRegistry()
	workflowRepo := memory.NewWorkflowRepository()
	eventPublisher := eventbus.New()
	policyEngine := policy.NewEngine(logger)
	auditLogger := governance.NewInMemoryAuditLogger()

	return orchestrator.New(orchestrator.Config{
		Logger:          logger,
		WorkflowRepo:    workflowRepo,
		EventPublisher:  eventPublisher,
		PolicyEvaluator: policyEngine,
		AuditLogger:     auditLogger,
		LLMRegistry:     llmRegistry,
		AgentRegistry:   agentRegistry,
	})
}
