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
	"github.com/felixgeelhaar/bridge/pkg/config"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

// RunCommand returns the run command.
func RunCommand() *cli.Command {
	return &cli.Command{
		Name:  "run",
		Usage: "Execute a workflow",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "workflow",
				Aliases:  []string{"w"},
				Usage:    "Path to workflow YAML file",
				Required: true,
			},
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Validate without executing",
			},
			&cli.StringSliceFlag{
				Name:  "input",
				Usage: "Input variables (key=value)",
			},
			&cli.StringFlag{
				Name:  "trigger-data",
				Usage: "JSON file with trigger data",
			},
			&cli.BoolFlag{
				Name:  "wait",
				Usage: "Wait for workflow to complete",
				Value: true,
			},
		},
		Action: runWorkflow,
	}
}

func runWorkflow(c *cli.Context) error {
	formatter := output.NewFormatter(c.String("output"))
	workflowPath := c.String("workflow")
	dryRun := c.Bool("dry-run")
	inputs := c.StringSlice("input")

	// Setup logger
	logLevel := c.String("log-level")
	logger := setupLogger(logLevel)

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

	// Validate first
	errors, warnings := validateWorkflow(&cfg, false)
	if len(errors) > 0 {
		formatter.ValidationResult(false, errors, warnings)
		return fmt.Errorf("validation failed")
	}

	for _, w := range warnings {
		formatter.Warning(w)
	}

	if dryRun {
		formatter.Success("Dry run: workflow is valid")
		formatter.WorkflowDefinition(nil) // Would need to create definition first
		return nil
	}

	// Build trigger data from inputs
	triggerData := make(map[string]any)
	for _, input := range inputs {
		key, value := parseInput(input)
		if key != "" {
			triggerData[key] = value
		}
	}

	// Initialize infrastructure
	ctx := context.Background()

	// Create LLM registry
	llmRegistry := llm.NewRegistry()

	// Try to setup providers from environment
	if err := setupProviders(llmRegistry, logger); err != nil {
		formatter.Warning(fmt.Sprintf("Failed to setup some providers: %v", err))
	}

	// Create agent registry with default agents
	agentRegistry := agents.NewAgentRegistry()
	for _, agent := range agents.DefaultAgents() {
		agentRegistry.Register(agent)
	}

	// Create repositories
	workflowRepo := memory.NewWorkflowRepository()
	eventPublisher := eventbus.New()

	// Create policy engine
	policyEngine := policy.NewEngine(logger)
	defaultBundle := &governance.PolicyBundle{
		Name:    "default",
		Version: "1.0",
		Active:  true,
		Rules: []governance.PolicyRule{
			{
				Name:     "default",
				Enabled:  true,
				Rego:     policy.DefaultPolicies(),
				Severity: governance.SeverityError,
			},
		},
	}
	policyEngine.LoadBundle(defaultBundle)

	// Create audit logger
	auditLogger := governance.NewInMemoryAuditLogger()

	// Create orchestrator
	orch, err := orchestrator.New(orchestrator.Config{
		Logger:          logger,
		WorkflowRepo:    workflowRepo,
		EventPublisher:  eventPublisher,
		PolicyEvaluator: policyEngine,
		AuditLogger:     auditLogger,
		LLMRegistry:     llmRegistry,
		AgentRegistry:   agentRegistry,
	})
	if err != nil {
		formatter.Error(fmt.Sprintf("Failed to create orchestrator: %v", err))
		return err
	}

	// Create workflow definition
	formatter.Info(fmt.Sprintf("Creating workflow: %s", cfg.Name))
	def, err := orch.CreateWorkflow(ctx, &cfg)
	if err != nil {
		formatter.Error(fmt.Sprintf("Failed to create workflow: %v", err))
		return err
	}

	// Create run
	formatter.Info("Starting workflow run...")
	run, err := orch.CreateRun(ctx, def, "cli", triggerData)
	if err != nil {
		formatter.Error(fmt.Sprintf("Failed to create run: %v", err))
		return err
	}

	formatter.Info(fmt.Sprintf("Run ID: %s", run.ID.String()))

	// Execute workflow
	err = orch.ExecuteWorkflow(ctx, run)
	if err != nil {
		// Check if it's awaiting approval
		if run.Status == "awaiting_approval" {
			formatter.Warning("Workflow is awaiting approval")
			formatter.Info(fmt.Sprintf("To approve: bridge approve %s", run.ID.String()))
			formatter.WorkflowRun(run)
			return nil
		}

		formatter.Error(fmt.Sprintf("Workflow execution failed: %v", err))
		formatter.WorkflowRun(run)
		return err
	}

	formatter.Success("Workflow completed successfully")
	formatter.WorkflowRun(run)

	return nil
}

func setupLogger(level string) *bolt.Logger {
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

func setupProviders(registry *llm.Registry, logger *bolt.Logger) error {
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
		logger.Info().Str("provider", "anthropic").Msg("Provider registered")
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
		logger.Info().Str("provider", "openai").Msg("Provider registered")
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
		logger.Info().Str("provider", "gemini").Msg("Provider registered")
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

func parseInput(input string) (string, string) {
	for i, c := range input {
		if c == '=' {
			return input[:i], input[i+1:]
		}
	}
	return input, ""
}
