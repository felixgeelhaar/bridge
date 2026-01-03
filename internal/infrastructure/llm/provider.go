package llm

import (
	"context"
	"time"
)

// Provider defines the interface for LLM providers.
type Provider interface {
	// Name returns the provider name (e.g., "anthropic", "openai").
	Name() string

	// Complete sends a completion request and returns the response.
	Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)

	// Models returns the list of available models.
	Models() []string
}

// CompletionRequest represents a request to an LLM.
type CompletionRequest struct {
	Model         string
	SystemPrompt  string
	Messages      []Message
	Tools         []Tool
	MaxTokens     int
	Temperature   float64
	TopP          float64
	StopSequences []string
	Metadata      map[string]any
}

// Message represents a chat message.
type Message struct {
	Role    Role
	Content string
	Name    string // Optional: function/tool name for tool responses
}

// Role represents the role of a message sender.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Tool represents a tool that can be called by the LLM.
type Tool struct {
	Name        string
	Description string
	Parameters  map[string]any // JSON Schema for parameters
}

// CompletionResponse represents a response from an LLM.
type CompletionResponse struct {
	Content      string
	ToolCalls    []ToolCall
	FinishReason FinishReason
	Usage        Usage
	Model        string
	Latency      time.Duration
}

// ToolCall represents a tool call request from the LLM.
type ToolCall struct {
	ID        string
	Name      string
	Arguments map[string]any
}

// FinishReason indicates why the completion finished.
type FinishReason string

const (
	FinishReasonStop      FinishReason = "stop"
	FinishReasonMaxTokens FinishReason = "max_tokens"
	FinishReasonToolUse   FinishReason = "tool_use"
	FinishReasonError     FinishReason = "error"
)

// Usage represents token usage information.
type Usage struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
}

// ProviderConfig contains common configuration for providers.
type ProviderConfig struct {
	APIKey      string
	BaseURL     string
	Model       string
	MaxTokens   int
	Temperature float64
	Timeout     time.Duration
}

// DefaultProviderConfig returns default provider configuration.
func DefaultProviderConfig() ProviderConfig {
	return ProviderConfig{
		MaxTokens:   4096,
		Temperature: 0.7,
		Timeout:     2 * time.Minute,
	}
}

// ProviderError represents an error from an LLM provider.
type ProviderError struct {
	Provider   string
	StatusCode int
	Message    string
	Retryable  bool
}

func (e *ProviderError) Error() string {
	return e.Provider + ": " + e.Message
}

// IsRetryable returns true if the error can be retried.
func (e *ProviderError) IsRetryable() bool {
	return e.Retryable
}

// NewProviderError creates a new provider error.
func NewProviderError(provider string, statusCode int, message string, retryable bool) *ProviderError {
	return &ProviderError{
		Provider:   provider,
		StatusCode: statusCode,
		Message:    message,
		Retryable:  retryable,
	}
}

// Registry manages multiple LLM providers.
type Registry struct {
	providers map[string]Provider
}

// NewRegistry creates a new provider registry.
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
	}
}

// Register adds a provider to the registry.
func (r *Registry) Register(provider Provider) {
	r.providers[provider.Name()] = provider
}

// Get retrieves a provider by name.
func (r *Registry) Get(name string) (Provider, bool) {
	p, ok := r.providers[name]
	return p, ok
}

// List returns all registered provider names.
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}
