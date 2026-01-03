package agents

import (
	"context"
	"time"

	"github.com/felixgeelhaar/bridge/pkg/types"
)

// AgentEntity represents a persisted agent configuration.
type AgentEntity struct {
	ID           types.AgentID
	Name         string
	Description  string
	Provider     string
	Model        string
	SystemPrompt string
	MaxTokens    int
	Temperature  float64
	Capabilities []string
	Metadata     map[string]any
	Active       bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ToAgent converts an entity to an Agent.
func (e *AgentEntity) ToAgent() *Agent {
	return &Agent{
		ID:           e.ID,
		Name:         e.Name,
		Description:  e.Description,
		Provider:     e.Provider,
		Model:        e.Model,
		SystemPrompt: e.SystemPrompt,
		MaxTokens:    e.MaxTokens,
		Temperature:  e.Temperature,
		Capabilities: e.Capabilities,
		Metadata:     e.Metadata,
	}
}

// NewAgentEntity creates an entity from an Agent.
func NewAgentEntity(agent *Agent) *AgentEntity {
	now := time.Now()
	return &AgentEntity{
		ID:           agent.ID,
		Name:         agent.Name,
		Description:  agent.Description,
		Provider:     agent.Provider,
		Model:        agent.Model,
		SystemPrompt: agent.SystemPrompt,
		MaxTokens:    agent.MaxTokens,
		Temperature:  agent.Temperature,
		Capabilities: agent.Capabilities,
		Metadata:     agent.Metadata,
		Active:       true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// Repository provides persistence for agent configurations.
type Repository interface {
	Create(ctx context.Context, agent *AgentEntity) error
	Get(ctx context.Context, id types.AgentID) (*AgentEntity, error)
	GetByName(ctx context.Context, name string) (*AgentEntity, error)
	List(ctx context.Context, limit, offset int) ([]*AgentEntity, error)
	ListActive(ctx context.Context) ([]*AgentEntity, error)
	Update(ctx context.Context, agent *AgentEntity) error
	Delete(ctx context.Context, id types.AgentID) error
}
