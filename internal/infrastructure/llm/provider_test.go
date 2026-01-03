package llm

import (
	"context"
	"testing"
)

func TestRegistry(t *testing.T) {
	registry := NewRegistry()

	// Initially empty
	if len(registry.List()) != 0 {
		t.Error("New registry should be empty")
	}

	// Create mock providers
	provider1 := &MockProvider{name: "provider1"}
	provider2 := &MockProvider{name: "provider2"}

	// Register providers
	registry.Register(provider1)
	registry.Register(provider2)

	// Check list
	list := registry.List()
	if len(list) != 2 {
		t.Errorf("List() returned %d providers, want 2", len(list))
	}

	// Get provider
	p, ok := registry.Get("provider1")
	if !ok {
		t.Error("Get() should return true for registered provider")
	}
	if p.Name() != "provider1" {
		t.Errorf("Get() returned wrong provider: %s", p.Name())
	}

	// Get non-existent
	_, ok = registry.Get("nonexistent")
	if ok {
		t.Error("Get() should return false for non-existent provider")
	}
}

func TestDefaultProviderConfig(t *testing.T) {
	cfg := DefaultProviderConfig()

	if cfg.MaxTokens == 0 {
		t.Error("MaxTokens should have a default value")
	}
	if cfg.Temperature == 0 {
		t.Error("Temperature should have a default value")
	}
	if cfg.Timeout == 0 {
		t.Error("Timeout should have a default value")
	}
}

func TestProviderError(t *testing.T) {
	err := NewProviderError("test", 500, "internal error", true)

	if err.Provider != "test" {
		t.Errorf("Provider = %v, want test", err.Provider)
	}
	if err.StatusCode != 500 {
		t.Errorf("StatusCode = %v, want 500", err.StatusCode)
	}
	if !err.IsRetryable() {
		t.Error("IsRetryable() should return true")
	}

	errStr := err.Error()
	if errStr == "" {
		t.Error("Error() should return non-empty string")
	}

	// Non-retryable error
	err2 := NewProviderError("test", 400, "bad request", false)
	if err2.IsRetryable() {
		t.Error("IsRetryable() should return false")
	}
}

func TestCompletionRequest(t *testing.T) {
	req := &CompletionRequest{
		Model:        "test-model",
		SystemPrompt: "You are a helpful assistant.",
		Messages: []Message{
			{Role: RoleUser, Content: "Hello"},
		},
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	if req.Model != "test-model" {
		t.Error("Model not set correctly")
	}
	if len(req.Messages) != 1 {
		t.Error("Messages not set correctly")
	}
	if req.Messages[0].Role != RoleUser {
		t.Error("Message role not set correctly")
	}
}

func TestCompletionResponse(t *testing.T) {
	resp := &CompletionResponse{
		Content:      "Hello! How can I help?",
		FinishReason: FinishReasonStop,
		Usage: Usage{
			InputTokens:  10,
			OutputTokens: 5,
			TotalTokens:  15,
		},
	}

	if resp.Content == "" {
		t.Error("Content should not be empty")
	}
	if resp.FinishReason != FinishReasonStop {
		t.Error("FinishReason not set correctly")
	}
	if resp.Usage.TotalTokens != 15 {
		t.Error("Usage not calculated correctly")
	}
}

func TestToolCall(t *testing.T) {
	tc := ToolCall{
		ID:   "call_123",
		Name: "search",
		Arguments: map[string]any{
			"query": "test query",
		},
	}

	if tc.ID != "call_123" {
		t.Error("ToolCall ID not set correctly")
	}
	if tc.Arguments["query"] != "test query" {
		t.Error("ToolCall arguments not set correctly")
	}
}

func TestFinishReason(t *testing.T) {
	tests := []struct {
		reason FinishReason
		want   string
	}{
		{FinishReasonStop, "stop"},
		{FinishReasonMaxTokens, "max_tokens"},
		{FinishReasonToolUse, "tool_use"},
		{FinishReasonError, "error"},
	}

	for _, tt := range tests {
		if string(tt.reason) != tt.want {
			t.Errorf("FinishReason %v = %v, want %v", tt.reason, string(tt.reason), tt.want)
		}
	}
}

func TestRole(t *testing.T) {
	tests := []struct {
		role Role
		want string
	}{
		{RoleSystem, "system"},
		{RoleUser, "user"},
		{RoleAssistant, "assistant"},
		{RoleTool, "tool"},
	}

	for _, tt := range tests {
		if string(tt.role) != tt.want {
			t.Errorf("Role %v = %v, want %v", tt.role, string(tt.role), tt.want)
		}
	}
}

// MockProvider is a mock implementation of Provider for testing.
type MockProvider struct {
	name         string
	models       []string
	completeFunc func(*CompletionRequest) (*CompletionResponse, error)
}

func (m *MockProvider) Name() string {
	return m.name
}

func (m *MockProvider) Models() []string {
	if m.models != nil {
		return m.models
	}
	return []string{"mock-model"}
}

func (m *MockProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	if m.completeFunc != nil {
		return m.completeFunc(req)
	}
	return &CompletionResponse{
		Content:      "Mock response",
		FinishReason: FinishReasonStop,
		Usage: Usage{
			InputTokens:  10,
			OutputTokens: 5,
			TotalTokens:  15,
		},
	}, nil
}

// Ensure MockProvider implements Provider
var _ Provider = (*MockProvider)(nil)
