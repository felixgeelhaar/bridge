package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/felixgeelhaar/bolt"
	"github.com/felixgeelhaar/bridge/internal/application/orchestrator"
	"github.com/felixgeelhaar/bridge/internal/domain/agents"
	"github.com/felixgeelhaar/bridge/internal/domain/governance"
	"github.com/felixgeelhaar/bridge/internal/domain/workflow"
	"github.com/felixgeelhaar/bridge/internal/infrastructure/llm"
	"github.com/felixgeelhaar/bridge/internal/infrastructure/messaging/eventbus"
	"github.com/felixgeelhaar/bridge/internal/infrastructure/persistence/memory"
	"github.com/felixgeelhaar/bridge/internal/infrastructure/policy"
	"github.com/felixgeelhaar/bridge/internal/interfaces/cli/output"
	"github.com/felixgeelhaar/bridge/pkg/types"
	"github.com/google/uuid"
	"github.com/urfave/cli/v2"
)

// ApproveCommand returns the approve command.
func ApproveCommand() *cli.Command {
	return &cli.Command{
		Name:      "approve",
		Usage:     "Approve a workflow run awaiting approval",
		ArgsUsage: "<run-id>",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "reject",
				Usage: "Reject instead of approve",
			},
			&cli.StringFlag{
				Name:    "comment",
				Aliases: []string{"m"},
				Usage:   "Comment or reason for approval/rejection",
			},
			&cli.StringFlag{
				Name:  "approver",
				Usage: "Approver identity",
				Value: getCurrentUser(),
			},
		},
		Action: runApprove,
	}
}

func runApprove(c *cli.Context) error {
	formatter := output.NewFormatter(c.String("output"))
	reject := c.Bool("reject")
	comment := c.String("comment")
	approver := c.String("approver")

	runID := c.Args().First()
	if runID == "" {
		formatter.Error("Run ID required")
		return fmt.Errorf("run id required")
	}

	// Parse run ID
	id, err := uuid.Parse(runID)
	if err != nil {
		formatter.Error(fmt.Sprintf("Invalid run ID: %s", runID))
		return err
	}

	// Setup infrastructure
	ctx := context.Background()
	logLevel := c.String("log-level")
	logger := setupApproveLogger(logLevel)

	// Create orchestrator
	orch, auditLogger, err := createApprovalOrchestrator(logger)
	if err != nil {
		formatter.Error(fmt.Sprintf("Failed to initialize: %v", err))
		return err
	}

	// Get run
	run, err := orch.GetRun(ctx, types.RunID(id.String()))
	if err != nil {
		formatter.Error(fmt.Sprintf("Run not found: %s", runID))
		return err
	}

	// Check status
	if run.Status != workflow.RunStatusAwaitingApproval {
		formatter.Error(fmt.Sprintf("Run is not awaiting approval (status: %s)", run.Status))
		return fmt.Errorf("run not awaiting approval")
	}

	// Create audit service
	auditService := governance.NewAuditService(auditLogger)

	if reject {
		// Reject the run
		run.Fail("Rejected by " + approver + ": " + comment)
		auditService.LogWorkflowFailed(ctx, run.WorkflowID.String(), run.ID.String(), "Rejected by "+approver)
		formatter.ApprovalStatus(run.ID.String(), "rejected", approver)
		return nil
	}

	// Approve and resume
	formatter.Info(fmt.Sprintf("Approving run %s...", runID[:8]))
	auditService.LogApprovalGranted(ctx, run.ID.String(), run.ID.String(), approver)

	// Resume workflow execution
	err = orch.ResumeWorkflow(ctx, run)
	if err != nil {
		formatter.Error(fmt.Sprintf("Failed to resume workflow: %v", err))
		return err
	}

	formatter.ApprovalStatus(run.ID.String(), "approved", approver)
	formatter.Success("Workflow resumed successfully")
	formatter.WorkflowRun(run)

	return nil
}

func setupApproveLogger(level string) *bolt.Logger {
	var logLevel bolt.Level
	switch level {
	case "debug":
		logLevel = bolt.DEBUG
	case "warn":
		logLevel = bolt.WARN
	case "error":
		logLevel = bolt.ERROR
	default:
		logLevel = bolt.INFO
	}

	handler := bolt.NewConsoleHandler(os.Stderr)
	return bolt.New(handler).SetLevel(logLevel)
}

func createApprovalOrchestrator(logger *bolt.Logger) (*orchestrator.Orchestrator, governance.AuditLogger, error) {
	llmRegistry := llm.NewRegistry()

	// Setup providers
	if err := setupProvidersForApproval(llmRegistry, logger); err != nil {
		logger.Warn().Err(err).Msg("Some providers failed to initialize")
	}

	agentRegistry := agents.NewAgentRegistry()
	for _, agent := range agents.DefaultAgents() {
		agentRegistry.Register(agent)
	}

	workflowRepo := memory.NewWorkflowRepository()
	eventPublisher := eventbus.New()
	policyEngine := policy.NewEngine(logger)
	auditLogger := governance.NewInMemoryAuditLogger()

	orch, err := orchestrator.New(orchestrator.Config{
		Logger:          logger,
		WorkflowRepo:    workflowRepo,
		EventPublisher:  eventPublisher,
		PolicyEvaluator: policyEngine,
		AuditLogger:     auditLogger,
		LLMRegistry:     llmRegistry,
		AgentRegistry:   agentRegistry,
	})

	return orch, auditLogger, err
}

func setupProvidersForApproval(registry *llm.Registry, logger *bolt.Logger) error {
	resilientCfg := llm.DefaultResilientConfig()
	rateLimitCfg := llm.DefaultRateLimitConfig()

	// Try Anthropic
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		provider := llm.NewAnthropicProvider(llm.AnthropicConfig{
			ProviderConfig: llm.ProviderConfig{
				APIKey: apiKey,
				Model:  "claude-sonnet-4-20250514",
			},
		})
		resilient := llm.NewResilientProvider(provider, resilientCfg)
		rateLimited := llm.NewRateLimitedProvider(resilient, logger, rateLimitCfg)
		registry.Register(rateLimited)
	}

	// Try OpenAI
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		provider := llm.NewOpenAIProvider(llm.OpenAIConfig{
			ProviderConfig: llm.ProviderConfig{
				APIKey: apiKey,
				Model:  "gpt-4o",
			},
		})
		resilient := llm.NewResilientProvider(provider, resilientCfg)
		rateLimited := llm.NewRateLimitedProvider(resilient, logger, rateLimitCfg)
		registry.Register(rateLimited)
	}

	// Try Gemini
	if apiKey := os.Getenv("GEMINI_API_KEY"); apiKey != "" {
		provider := llm.NewGeminiProvider(llm.GeminiConfig{
			ProviderConfig: llm.ProviderConfig{
				APIKey: apiKey,
				Model:  "gemini-1.5-pro",
			},
		})
		resilient := llm.NewResilientProvider(provider, resilientCfg)
		rateLimited := llm.NewRateLimitedProvider(resilient, logger, rateLimitCfg)
		registry.Register(rateLimited)
	}

	// Try Ollama
	baseURL := os.Getenv("OLLAMA_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	provider := llm.NewOllamaProvider(llm.OllamaConfig{
		ProviderConfig: llm.ProviderConfig{
			BaseURL: baseURL,
			Model:   "llama3",
		},
	})
	resilient := llm.NewResilientProvider(provider, resilientCfg)
	registry.Register(resilient)

	return nil
}

func getCurrentUser() string {
	if user := os.Getenv("USER"); user != "" {
		return user
	}
	if user := os.Getenv("USERNAME"); user != "" {
		return user
	}
	return "unknown"
}
