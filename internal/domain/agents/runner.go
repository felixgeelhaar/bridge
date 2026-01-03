package agents

import (
	"context"
	"fmt"
	"time"

	"github.com/felixgeelhaar/bolt"
	"github.com/felixgeelhaar/bridge/internal/infrastructure/llm"
	"github.com/felixgeelhaar/bridge/pkg/types"
)

// Agent represents an AI agent configuration.
type Agent struct {
	ID           types.AgentID
	Name         string
	Description  string
	Provider     string // anthropic, openai, gemini, ollama
	Model        string
	SystemPrompt string
	Tools        []llm.Tool
	MaxTokens    int
	Temperature  float64
	Capabilities []string
	Metadata     map[string]any
}

// AgentResponse represents the result of an agent invocation.
type AgentResponse struct {
	Content      string
	ToolCalls    []llm.ToolCall
	TokensIn     int
	TokensOut    int
	Duration     time.Duration
	Model        string
	FinishReason llm.FinishReason
}

// Runner executes agent calls.
type Runner interface {
	Execute(ctx context.Context, agent *Agent, messages []llm.Message) (*AgentResponse, error)
}

// runner implements the Runner interface.
type runner struct {
	logger   *bolt.Logger
	registry *llm.Registry
}

// NewRunner creates a new agent runner.
func NewRunner(logger *bolt.Logger, registry *llm.Registry) Runner {
	return &runner{
		logger:   logger,
		registry: registry,
	}
}

// Execute invokes an agent with the given messages.
func (r *runner) Execute(ctx context.Context, agent *Agent, messages []llm.Message) (*AgentResponse, error) {
	start := time.Now()

	// Get provider
	provider, ok := r.registry.Get(agent.Provider)
	if !ok {
		return nil, fmt.Errorf("%w: %s", types.ErrLLMProviderNotFound, agent.Provider)
	}

	// Log with context
	logger := r.logger.Ctx(ctx)
	logger.Info().
		Str("agent_id", agent.ID.String()).
		Str("agent_name", agent.Name).
		Str("provider", agent.Provider).
		Str("model", agent.Model).
		Int("messages", len(messages)).
		Msg("Executing agent")

	// Build completion request
	req := &llm.CompletionRequest{
		Model:        agent.Model,
		SystemPrompt: agent.SystemPrompt,
		Messages:     messages,
		Tools:        agent.Tools,
		MaxTokens:    agent.MaxTokens,
		Temperature:  agent.Temperature,
	}

	// Execute completion
	resp, err := provider.Complete(ctx, req)
	if err != nil {
		logger.Error().
			Err(err).
			Str("agent_id", agent.ID.String()).
			Dur("duration", time.Since(start)).
			Msg("Agent execution failed")
		return nil, err
	}

	result := &AgentResponse{
		Content:      resp.Content,
		ToolCalls:    resp.ToolCalls,
		TokensIn:     resp.Usage.InputTokens,
		TokensOut:    resp.Usage.OutputTokens,
		Duration:     resp.Latency,
		Model:        resp.Model,
		FinishReason: resp.FinishReason,
	}

	logger.Info().
		Str("agent_id", agent.ID.String()).
		Int("tokens_in", result.TokensIn).
		Int("tokens_out", result.TokensOut).
		Dur("duration", result.Duration).
		Str("finish_reason", string(result.FinishReason)).
		Int("tool_calls", len(result.ToolCalls)).
		Msg("Agent execution completed")

	return result, nil
}

// AgentRegistry manages agent configurations.
type AgentRegistry struct {
	agents map[string]*Agent
}

// NewAgentRegistry creates a new agent registry.
func NewAgentRegistry() *AgentRegistry {
	return &AgentRegistry{
		agents: make(map[string]*Agent),
	}
}

// Register adds an agent to the registry.
func (r *AgentRegistry) Register(agent *Agent) {
	r.agents[agent.Name] = agent
}

// Get retrieves an agent by name.
func (r *AgentRegistry) Get(name string) (*Agent, bool) {
	agent, ok := r.agents[name]
	return agent, ok
}

// List returns all registered agent names.
func (r *AgentRegistry) List() []string {
	names := make([]string, 0, len(r.agents))
	for name := range r.agents {
		names = append(names, name)
	}
	return names
}

// DefaultAgents returns a set of default agent configurations.
func DefaultAgents() []*Agent {
	return []*Agent{
		{
			ID:          types.NewAgentID(),
			Name:        "code-reviewer",
			Description: "Reviews code changes for quality, security, and best practices",
			Provider:    "anthropic",
			Model:       "claude-sonnet-4-20250514",
			SystemPrompt: `You are an expert code reviewer. Analyze code changes for:
- Code quality and readability
- Security vulnerabilities
- Performance issues
- Best practices adherence
- Potential bugs

Provide specific, actionable feedback with code examples where helpful.`,
			MaxTokens:    4096,
			Temperature:  0.3,
			Capabilities: []string{"code-review", "security-analysis"},
		},
		{
			ID:          types.NewAgentID(),
			Name:        "security-analyst",
			Description: "Analyzes code for security vulnerabilities",
			Provider:    "anthropic",
			Model:       "claude-sonnet-4-20250514",
			SystemPrompt: `You are a security expert. Analyze code for:
- OWASP Top 10 vulnerabilities
- Injection flaws
- Authentication/authorization issues
- Sensitive data exposure
- Security misconfigurations

Provide severity ratings and remediation steps.`,
			MaxTokens:    4096,
			Temperature:  0.2,
			Capabilities: []string{"security-analysis"},
		},
		{
			ID:          types.NewAgentID(),
			Name:        "code-generator",
			Description: "Generates code based on requirements",
			Provider:    "anthropic",
			Model:       "claude-sonnet-4-20250514",
			SystemPrompt: `You are an expert software engineer. Generate clean, well-documented code that:
- Follows best practices
- Includes proper error handling
- Is well-tested
- Follows the project's coding style`,
			MaxTokens:    8192,
			Temperature:  0.5,
			Capabilities: []string{"code-generation"},
		},
	}
}
