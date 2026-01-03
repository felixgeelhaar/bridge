package agents

import (
	"context"
	"os"
	"testing"

	"github.com/felixgeelhaar/bolt"
	"github.com/felixgeelhaar/bridge/internal/infrastructure/llm"
	"github.com/felixgeelhaar/bridge/pkg/types"
)

func TestNewAgentRegistry(t *testing.T) {
	registry := NewAgentRegistry()
	if registry == nil {
		t.Fatal("NewAgentRegistry returned nil")
	}

	// Should be empty initially
	if len(registry.List()) != 0 {
		t.Error("New registry should be empty")
	}
}

func TestAgentRegistry_Register(t *testing.T) {
	registry := NewAgentRegistry()

	agent := &Agent{
		ID:          types.NewAgentID(),
		Name:        "test-agent",
		Description: "A test agent",
		Provider:    "anthropic",
		Model:       "claude-sonnet-4-20250514",
	}

	registry.Register(agent)

	// Should have 1 agent
	if len(registry.List()) != 1 {
		t.Errorf("Registry should have 1 agent, got %d", len(registry.List()))
	}
}

func TestAgentRegistry_Get(t *testing.T) {
	registry := NewAgentRegistry()

	agent := &Agent{
		ID:          types.NewAgentID(),
		Name:        "test-agent",
		Description: "A test agent",
	}

	registry.Register(agent)

	// Get existing agent
	retrieved, ok := registry.Get("test-agent")
	if !ok {
		t.Error("Get() should return true for existing agent")
	}
	if retrieved.Name != "test-agent" {
		t.Errorf("Got wrong agent: %s", retrieved.Name)
	}

	// Get non-existent agent
	_, ok = registry.Get("nonexistent")
	if ok {
		t.Error("Get() should return false for non-existent agent")
	}
}

func TestAgentRegistry_List(t *testing.T) {
	registry := NewAgentRegistry()

	registry.Register(&Agent{Name: "agent1"})
	registry.Register(&Agent{Name: "agent2"})
	registry.Register(&Agent{Name: "agent3"})

	list := registry.List()
	if len(list) != 3 {
		t.Errorf("List() returned %d agents, want 3", len(list))
	}

	// Check all agents are in list
	names := make(map[string]bool)
	for _, name := range list {
		names[name] = true
	}

	for _, expected := range []string{"agent1", "agent2", "agent3"} {
		if !names[expected] {
			t.Errorf("Agent %s not in list", expected)
		}
	}
}

func TestDefaultAgents(t *testing.T) {
	agents := DefaultAgents()

	if len(agents) == 0 {
		t.Fatal("DefaultAgents() should return at least one agent")
	}

	// Check each agent has required fields
	for _, agent := range agents {
		if agent.Name == "" {
			t.Error("Agent name should not be empty")
		}
		if agent.Description == "" {
			t.Error("Agent description should not be empty")
		}
		if agent.Provider == "" {
			t.Error("Agent provider should not be empty")
		}
		if agent.Model == "" {
			t.Error("Agent model should not be empty")
		}
		if agent.SystemPrompt == "" {
			t.Error("Agent system prompt should not be empty")
		}
		if agent.ID == "" {
			t.Error("Agent ID should not be empty")
		}
	}

	// Check specific agents exist
	agentNames := make(map[string]bool)
	for _, agent := range agents {
		agentNames[agent.Name] = true
	}

	expectedAgents := []string{"code-reviewer", "security-analyst", "code-generator"}
	for _, name := range expectedAgents {
		if !agentNames[name] {
			t.Errorf("Expected agent %s not found in defaults", name)
		}
	}
}

func TestAgent_Fields(t *testing.T) {
	agent := &Agent{
		ID:           types.NewAgentID(),
		Name:         "test-agent",
		Description:  "Test description",
		Provider:     "anthropic",
		Model:        "claude-sonnet-4-20250514",
		SystemPrompt: "You are a helpful assistant.",
		MaxTokens:    4096,
		Temperature:  0.7,
		Capabilities: []string{"capability1", "capability2"},
		Metadata: map[string]any{
			"key": "value",
		},
	}

	if agent.ID == "" {
		t.Error("ID should not be empty")
	}
	if agent.Name != "test-agent" {
		t.Error("Name not set correctly")
	}
	if agent.Provider != "anthropic" {
		t.Error("Provider not set correctly")
	}
	if agent.MaxTokens != 4096 {
		t.Error("MaxTokens not set correctly")
	}
	if agent.Temperature != 0.7 {
		t.Error("Temperature not set correctly")
	}
	if len(agent.Capabilities) != 2 {
		t.Error("Capabilities not set correctly")
	}
	if agent.Metadata["key"] != "value" {
		t.Error("Metadata not set correctly")
	}
}

func TestAgentResponse_Fields(t *testing.T) {
	resp := &AgentResponse{
		Content:      "Hello, world!",
		TokensIn:     100,
		TokensOut:    50,
		Model:        "claude-sonnet-4-20250514",
		FinishReason: llm.FinishReasonStop,
	}

	if resp.Content != "Hello, world!" {
		t.Error("Content not set correctly")
	}
	if resp.TokensIn != 100 {
		t.Error("TokensIn not set correctly")
	}
	if resp.TokensOut != 50 {
		t.Error("TokensOut not set correctly")
	}
	if resp.FinishReason != llm.FinishReasonStop {
		t.Error("FinishReason not set correctly")
	}
}

func TestNewRunner(t *testing.T) {
	handler := bolt.NewConsoleHandler(os.Stderr)
	logger := bolt.New(handler)
	registry := llm.NewRegistry()

	runner := NewRunner(logger, registry)
	if runner == nil {
		t.Fatal("NewRunner returned nil")
	}
}

func TestRunner_Execute_ProviderNotFound(t *testing.T) {
	handler := bolt.NewConsoleHandler(os.Stderr)
	logger := bolt.New(handler).SetLevel(bolt.ERROR)
	registry := llm.NewRegistry()
	runner := NewRunner(logger, registry)

	agent := &Agent{
		ID:       types.NewAgentID(),
		Name:     "test-agent",
		Provider: "nonexistent-provider",
		Model:    "test-model",
	}

	_, err := runner.Execute(context.Background(), agent, nil)
	if err == nil {
		t.Error("Execute should return error for non-existent provider")
	}
}

func TestAgentRegistry_OverwriteAgent(t *testing.T) {
	registry := NewAgentRegistry()

	agent1 := &Agent{
		ID:          types.NewAgentID(),
		Name:        "test-agent",
		Description: "First version",
	}

	agent2 := &Agent{
		ID:          types.NewAgentID(),
		Name:        "test-agent",
		Description: "Second version",
	}

	registry.Register(agent1)
	registry.Register(agent2)

	// Should still have only 1 agent (overwritten)
	if len(registry.List()) != 1 {
		t.Errorf("Registry should have 1 agent after overwrite, got %d", len(registry.List()))
	}

	// Should get the second version
	retrieved, _ := registry.Get("test-agent")
	if retrieved.Description != "Second version" {
		t.Error("Agent should be overwritten with second version")
	}
}
