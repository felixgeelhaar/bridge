package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/felixgeelhaar/bolt"
	"github.com/felixgeelhaar/bridge/internal/domain/agents"
	"github.com/felixgeelhaar/bridge/internal/infrastructure/persistence/postgres/sqlc"
	"github.com/felixgeelhaar/bridge/pkg/types"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AgentRepository implements agents.Repository using PostgreSQL.
type AgentRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
	logger  *bolt.Logger
}

// NewAgentRepository creates a new PostgreSQL agent repository.
func NewAgentRepository(pool *pgxpool.Pool, logger *bolt.Logger) *AgentRepository {
	return &AgentRepository{
		pool:    pool,
		queries: sqlc.New(pool),
		logger:  logger,
	}
}

// Create creates a new agent.
func (r *AgentRepository) Create(ctx context.Context, agent *agents.AgentEntity) error {
	metadata, err := json.Marshal(agent.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = r.queries.CreateAgent(ctx, sqlc.CreateAgentParams{
		ID:           agent.ID.String(),
		Name:         agent.Name,
		Description:  strPtr(agent.Description),
		Provider:     agent.Provider,
		Model:        agent.Model,
		SystemPrompt: strPtr(agent.SystemPrompt),
		MaxTokens:    int32Ptr(int32(agent.MaxTokens)),
		Temperature:  floatToNumeric(agent.Temperature),
		Capabilities: agent.Capabilities,
		Metadata:     metadata,
		Active:       agent.Active,
		CreatedAt:    timeToPgTimestamptzValue(agent.CreatedAt),
		UpdatedAt:    timeToPgTimestamptzValue(agent.UpdatedAt),
	})
	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}

	r.logger.Debug().
		Str("agent_id", agent.ID.String()).
		Str("name", agent.Name).
		Msg("Created agent")

	return nil
}

// Get retrieves an agent by ID.
func (r *AgentRepository) Get(ctx context.Context, id types.AgentID) (*agents.AgentEntity, error) {
	row, err := r.queries.GetAgent(ctx, id.String())
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("agent not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	return r.rowToAgent(row)
}

// GetByName retrieves an agent by name.
func (r *AgentRepository) GetByName(ctx context.Context, name string) (*agents.AgentEntity, error) {
	row, err := r.queries.GetAgentByName(ctx, name)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("agent not found: %s", name)
		}
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	return r.rowToAgent(row)
}

// List lists agents with pagination.
func (r *AgentRepository) List(ctx context.Context, limit, offset int) ([]*agents.AgentEntity, error) {
	rows, err := r.queries.ListAgents(ctx, sqlc.ListAgentsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}

	result := make([]*agents.AgentEntity, 0, len(rows))
	for _, row := range rows {
		agent, err := r.rowToAgent(row)
		if err != nil {
			return nil, err
		}
		result = append(result, agent)
	}

	return result, nil
}

// ListActive lists all active agents.
func (r *AgentRepository) ListActive(ctx context.Context) ([]*agents.AgentEntity, error) {
	rows, err := r.queries.ListActiveAgents(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list active agents: %w", err)
	}

	result := make([]*agents.AgentEntity, 0, len(rows))
	for _, row := range rows {
		agent, err := r.rowToAgent(row)
		if err != nil {
			return nil, err
		}
		result = append(result, agent)
	}

	return result, nil
}

// Update updates an agent.
func (r *AgentRepository) Update(ctx context.Context, agent *agents.AgentEntity) error {
	metadata, err := json.Marshal(agent.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = r.queries.UpdateAgent(ctx, sqlc.UpdateAgentParams{
		ID:           agent.ID.String(),
		Name:         agent.Name,
		Description:  strPtr(agent.Description),
		Provider:     agent.Provider,
		Model:        agent.Model,
		SystemPrompt: strPtr(agent.SystemPrompt),
		MaxTokens:    int32Ptr(int32(agent.MaxTokens)),
		Temperature:  floatToNumeric(agent.Temperature),
		Capabilities: agent.Capabilities,
		Metadata:     metadata,
		Active:       agent.Active,
	})
	if err != nil {
		return fmt.Errorf("failed to update agent: %w", err)
	}

	return nil
}

// Delete deletes an agent.
func (r *AgentRepository) Delete(ctx context.Context, id types.AgentID) error {
	if err := r.queries.DeleteAgent(ctx, id.String()); err != nil {
		return fmt.Errorf("failed to delete agent: %w", err)
	}
	return nil
}

func (r *AgentRepository) rowToAgent(row sqlc.Agent) (*agents.AgentEntity, error) {
	var metadata map[string]any
	if len(row.Metadata) > 0 {
		if err := json.Unmarshal(row.Metadata, &metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &agents.AgentEntity{
		ID:           types.AgentID(row.ID),
		Name:         row.Name,
		Description:  ptrStr(row.Description),
		Provider:     row.Provider,
		Model:        row.Model,
		SystemPrompt: ptrStr(row.SystemPrompt),
		MaxTokens:    int(ptrInt32(row.MaxTokens)),
		Temperature:  numericToFloat(row.Temperature),
		Capabilities: row.Capabilities,
		Metadata:     metadata,
		Active:       row.Active,
		CreatedAt:    pgTimestamptzToTime(row.CreatedAt),
		UpdatedAt:    pgTimestamptzToTime(row.UpdatedAt),
	}, nil
}

func floatToNumeric(f float64) pgtype.Numeric {
	var n pgtype.Numeric
	n.Scan(fmt.Sprintf("%.2f", f))
	return n
}

func numericToFloat(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0.0
	}
	f, _ := n.Float64Value()
	return f.Float64
}

// Ensure AgentRepository implements agents.Repository.
var _ agents.Repository = (*AgentRepository)(nil)
