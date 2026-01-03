package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/felixgeelhaar/bolt"
	"github.com/felixgeelhaar/bridge/internal/domain/workflow"
	"github.com/felixgeelhaar/bridge/internal/infrastructure/persistence/postgres/sqlc"
	"github.com/felixgeelhaar/bridge/pkg/types"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WorkflowRepository implements workflow.Repository using PostgreSQL.
type WorkflowRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
	logger  *bolt.Logger
}

// NewWorkflowRepository creates a new PostgreSQL workflow repository.
func NewWorkflowRepository(pool *pgxpool.Pool, logger *bolt.Logger) *WorkflowRepository {
	return &WorkflowRepository{
		pool:    pool,
		queries: sqlc.New(pool),
		logger:  logger,
	}
}

// CreateDefinition creates a new workflow definition.
func (r *WorkflowRepository) CreateDefinition(ctx context.Context, def *workflow.WorkflowDefinition) error {
	config, err := r.marshalConfig(def)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	metadata, err := json.Marshal(def.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = r.queries.CreateWorkflowDefinition(ctx, sqlc.CreateWorkflowDefinitionParams{
		ID:          def.ID.String(),
		Name:        def.Name,
		Version:     def.Version,
		Description: strPtr(def.Description),
		Config:      config,
		Checksum:    strPtr(def.Checksum),
		Metadata:    metadata,
		CreatedAt:   timeToPgTimestamptzValue(def.CreatedAt),
		UpdatedAt:   timeToPgTimestamptzValue(def.UpdatedAt),
	})
	if err != nil {
		return fmt.Errorf("failed to create workflow definition: %w", err)
	}

	r.logger.Debug().
		Str("workflow_id", def.ID.String()).
		Str("name", def.Name).
		Msg("Created workflow definition")

	return nil
}

// GetDefinition retrieves a workflow definition by ID.
func (r *WorkflowRepository) GetDefinition(ctx context.Context, id types.WorkflowID) (*workflow.WorkflowDefinition, error) {
	row, err := r.queries.GetWorkflowDefinition(ctx, id.String())
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("workflow definition not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get workflow definition: %w", err)
	}

	return r.rowToDefinition(row)
}

// GetDefinitionByName retrieves a workflow definition by name.
func (r *WorkflowRepository) GetDefinitionByName(ctx context.Context, name string) (*workflow.WorkflowDefinition, error) {
	row, err := r.queries.GetWorkflowDefinitionByName(ctx, name)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("workflow definition not found: %s", name)
		}
		return nil, fmt.Errorf("failed to get workflow definition: %w", err)
	}

	return r.rowToDefinition(row)
}

// ListDefinitions lists workflow definitions with pagination.
func (r *WorkflowRepository) ListDefinitions(ctx context.Context, limit, offset int) ([]*workflow.WorkflowDefinition, error) {
	rows, err := r.queries.ListWorkflowDefinitions(ctx, sqlc.ListWorkflowDefinitionsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list workflow definitions: %w", err)
	}

	defs := make([]*workflow.WorkflowDefinition, 0, len(rows))
	for _, row := range rows {
		def, err := r.rowToDefinition(row)
		if err != nil {
			return nil, err
		}
		defs = append(defs, def)
	}

	return defs, nil
}

// UpdateDefinition updates a workflow definition.
func (r *WorkflowRepository) UpdateDefinition(ctx context.Context, def *workflow.WorkflowDefinition) error {
	config, err := r.marshalConfig(def)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	metadata, err := json.Marshal(def.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = r.queries.UpdateWorkflowDefinition(ctx, sqlc.UpdateWorkflowDefinitionParams{
		ID:          def.ID.String(),
		Name:        def.Name,
		Version:     def.Version,
		Description: strPtr(def.Description),
		Config:      config,
		Checksum:    strPtr(def.Checksum),
		Metadata:    metadata,
	})
	if err != nil {
		return fmt.Errorf("failed to update workflow definition: %w", err)
	}

	return nil
}

// DeleteDefinition deletes a workflow definition.
func (r *WorkflowRepository) DeleteDefinition(ctx context.Context, id types.WorkflowID) error {
	if err := r.queries.DeleteWorkflowDefinition(ctx, id.String()); err != nil {
		return fmt.Errorf("failed to delete workflow definition: %w", err)
	}
	return nil
}

// CreateRun creates a new workflow run with its steps.
func (r *WorkflowRepository) CreateRun(ctx context.Context, run *workflow.WorkflowRun) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := r.queries.WithTx(tx)

	runContext, err := json.Marshal(run.Context)
	if err != nil {
		return fmt.Errorf("failed to marshal context: %w", err)
	}

	triggerData, err := json.Marshal(run.TriggerData)
	if err != nil {
		return fmt.Errorf("failed to marshal trigger data: %w", err)
	}

	_, err = qtx.CreateWorkflowRun(ctx, sqlc.CreateWorkflowRunParams{
		ID:               run.ID.String(),
		WorkflowID:       run.WorkflowID.String(),
		WorkflowName:     run.WorkflowName,
		WorkflowVersion:  run.WorkflowVersion,
		Status:           string(run.Status),
		CurrentStepIndex: int32(run.CurrentStepIdx),
		Context:          runContext,
		TriggeredBy:      strPtr(run.TriggeredBy),
		TriggerData:      triggerData,
		Error:            strPtr(run.Error),
		StartedAt:        timeToPgTimestamptz(run.StartedAt),
		CompletedAt:      timeToPgTimestamptz(run.CompletedAt),
		CreatedAt:        timeToPgTimestamptzValue(run.CreatedAt),
		UpdatedAt:        timeToPgTimestamptzValue(run.UpdatedAt),
	})
	if err != nil {
		return fmt.Errorf("failed to create workflow run: %w", err)
	}

	// Create step runs
	for i, step := range run.Steps {
		input, _ := json.Marshal(step.Input)
		output, _ := json.Marshal(step.Output)

		_, err = qtx.CreateStepRun(ctx, sqlc.CreateStepRunParams{
			ID:               step.ID.String(),
			RunID:            run.ID.String(),
			StepIndex:        int32(step.StepIndex),
			Name:             step.Name,
			AgentID:          strPtr(step.AgentID),
			Status:           string(step.Status),
			Input:            input,
			Output:           output,
			RequiresApproval: step.RequiresApproval,
			TimeoutSeconds:   int32Ptr(int32(step.Timeout.Seconds())),
			MaxRetries:       int32Ptr(int32(step.MaxRetries)),
			RetryCount:       int32Ptr(int32(step.RetryCount)),
			Error:            strPtr(step.Error),
			TokensIn:         int32Ptr(int32(step.TokensIn)),
			TokensOut:        int32Ptr(int32(step.TokensOut)),
			StepOrder:        int32(i),
			StartedAt:        timeToPgTimestamptz(step.StartedAt),
			CompletedAt:      timeToPgTimestamptz(step.CompletedAt),
			CreatedAt:        timeToPgTimestamptzValue(step.CreatedAt),
		})
		if err != nil {
			return fmt.Errorf("failed to create step run: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.logger.Debug().
		Str("run_id", run.ID.String()).
		Str("workflow_name", run.WorkflowName).
		Msg("Created workflow run")

	return nil
}

// GetRun retrieves a workflow run by ID.
func (r *WorkflowRepository) GetRun(ctx context.Context, id types.RunID) (*workflow.WorkflowRun, error) {
	row, err := r.queries.GetWorkflowRun(ctx, id.String())
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("workflow run not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get workflow run: %w", err)
	}

	run, err := r.rowToRun(row)
	if err != nil {
		return nil, err
	}

	// Fetch steps
	stepRows, err := r.queries.ListStepRunsByRunID(ctx, id.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get step runs: %w", err)
	}

	run.Steps = make([]*workflow.StepRun, 0, len(stepRows))
	for _, stepRow := range stepRows {
		step, err := r.rowToStep(stepRow)
		if err != nil {
			return nil, err
		}
		run.Steps = append(run.Steps, step)
	}

	return run, nil
}

// ListRuns lists workflow runs for a given workflow.
func (r *WorkflowRepository) ListRuns(ctx context.Context, workflowID types.WorkflowID, limit, offset int) ([]*workflow.WorkflowRun, error) {
	rows, err := r.queries.ListWorkflowRuns(ctx, sqlc.ListWorkflowRunsParams{
		WorkflowID: workflowID.String(),
		Limit:      int32(limit),
		Offset:     int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list workflow runs: %w", err)
	}

	runs := make([]*workflow.WorkflowRun, 0, len(rows))
	for _, row := range rows {
		run, err := r.rowToRun(row)
		if err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}

	return runs, nil
}

// ListActiveRuns returns all active workflow runs.
func (r *WorkflowRepository) ListActiveRuns(ctx context.Context) ([]*workflow.WorkflowRun, error) {
	rows, err := r.queries.ListActiveWorkflowRuns(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list active workflow runs: %w", err)
	}

	runs := make([]*workflow.WorkflowRun, 0, len(rows))
	for _, row := range rows {
		run, err := r.rowToRun(row)
		if err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}

	return runs, nil
}

// UpdateRun updates a workflow run.
func (r *WorkflowRepository) UpdateRun(ctx context.Context, run *workflow.WorkflowRun) error {
	runContext, err := json.Marshal(run.Context)
	if err != nil {
		return fmt.Errorf("failed to marshal context: %w", err)
	}

	_, err = r.queries.UpdateWorkflowRun(ctx, sqlc.UpdateWorkflowRunParams{
		ID:               run.ID.String(),
		Status:           string(run.Status),
		CurrentStepIndex: int32(run.CurrentStepIdx),
		Context:          runContext,
		Error:            strPtr(run.Error),
		StartedAt:        timeToPgTimestamptz(run.StartedAt),
		CompletedAt:      timeToPgTimestamptz(run.CompletedAt),
	})
	if err != nil {
		return fmt.Errorf("failed to update workflow run: %w", err)
	}

	return nil
}

// GetStep retrieves a step run by ID.
func (r *WorkflowRepository) GetStep(ctx context.Context, id types.StepID) (*workflow.StepRun, error) {
	row, err := r.queries.GetStepRun(ctx, id.String())
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("step run not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get step run: %w", err)
	}

	return r.rowToStep(row)
}

// UpdateStep updates a step run.
func (r *WorkflowRepository) UpdateStep(ctx context.Context, step *workflow.StepRun) error {
	input, err := json.Marshal(step.Input)
	if err != nil {
		return fmt.Errorf("failed to marshal input: %w", err)
	}

	output, err := json.Marshal(step.Output)
	if err != nil {
		return fmt.Errorf("failed to marshal output: %w", err)
	}

	_, err = r.queries.UpdateStepRun(ctx, sqlc.UpdateStepRunParams{
		ID:          step.ID.String(),
		Status:      string(step.Status),
		Input:       input,
		Output:      output,
		RetryCount:  int32Ptr(int32(step.RetryCount)),
		Error:       strPtr(step.Error),
		TokensIn:    int32Ptr(int32(step.TokensIn)),
		TokensOut:   int32Ptr(int32(step.TokensOut)),
		StartedAt:   timeToPgTimestamptz(step.StartedAt),
		CompletedAt: timeToPgTimestamptz(step.CompletedAt),
	})
	if err != nil {
		return fmt.Errorf("failed to update step run: %w", err)
	}

	return nil
}

// Helper methods

func (r *WorkflowRepository) marshalConfig(def *workflow.WorkflowDefinition) ([]byte, error) {
	config := map[string]any{
		"steps":    def.Steps,
		"triggers": def.Triggers,
		"policies": def.Policies,
	}
	return json.Marshal(config)
}

func (r *WorkflowRepository) rowToDefinition(row sqlc.WorkflowDefinition) (*workflow.WorkflowDefinition, error) {
	var steps []workflow.StepDefinition
	var triggers []workflow.Trigger
	var policies []workflow.PolicyRef
	var metadata map[string]any

	if len(row.Config) > 0 {
		var config struct {
			Steps    []workflow.StepDefinition `json:"steps"`
			Triggers []workflow.Trigger        `json:"triggers"`
			Policies []workflow.PolicyRef      `json:"policies"`
		}
		if err := json.Unmarshal(row.Config, &config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %w", err)
		}
		steps = config.Steps
		triggers = config.Triggers
		policies = config.Policies
	}

	if len(row.Metadata) > 0 {
		if err := json.Unmarshal(row.Metadata, &metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &workflow.WorkflowDefinition{
		ID:          types.WorkflowID(row.ID),
		Name:        row.Name,
		Version:     row.Version,
		Description: ptrStr(row.Description),
		Steps:       steps,
		Triggers:    triggers,
		Policies:    policies,
		Checksum:    ptrStr(row.Checksum),
		Metadata:    metadata,
		CreatedAt:   pgTimestamptzToTime(row.CreatedAt),
		UpdatedAt:   pgTimestamptzToTime(row.UpdatedAt),
	}, nil
}

func (r *WorkflowRepository) rowToRun(row sqlc.WorkflowRun) (*workflow.WorkflowRun, error) {
	var runContext map[string]any
	var triggerData map[string]any

	if len(row.Context) > 0 {
		if err := json.Unmarshal(row.Context, &runContext); err != nil {
			return nil, fmt.Errorf("failed to unmarshal context: %w", err)
		}
	}

	if len(row.TriggerData) > 0 {
		if err := json.Unmarshal(row.TriggerData, &triggerData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal trigger data: %w", err)
		}
	}

	return &workflow.WorkflowRun{
		ID:              types.RunID(row.ID),
		WorkflowID:      types.WorkflowID(row.WorkflowID),
		WorkflowName:    row.WorkflowName,
		WorkflowVersion: row.WorkflowVersion,
		Status:          workflow.RunStatus(row.Status),
		CurrentStepIdx:  int(row.CurrentStepIndex),
		Context:         runContext,
		TriggeredBy:     ptrStr(row.TriggeredBy),
		TriggerData:     triggerData,
		Error:           ptrStr(row.Error),
		StartedAt:       pgTimestamptzToTimePtr(row.StartedAt),
		CompletedAt:     pgTimestamptzToTimePtr(row.CompletedAt),
		CreatedAt:       pgTimestamptzToTime(row.CreatedAt),
		UpdatedAt:       pgTimestamptzToTime(row.UpdatedAt),
	}, nil
}

func (r *WorkflowRepository) rowToStep(row sqlc.StepRun) (*workflow.StepRun, error) {
	var input map[string]any
	var output map[string]any

	if len(row.Input) > 0 {
		if err := json.Unmarshal(row.Input, &input); err != nil {
			return nil, fmt.Errorf("failed to unmarshal input: %w", err)
		}
	}

	if len(row.Output) > 0 {
		if err := json.Unmarshal(row.Output, &output); err != nil {
			return nil, fmt.Errorf("failed to unmarshal output: %w", err)
		}
	}

	return &workflow.StepRun{
		ID:               types.StepID(row.ID),
		RunID:            types.RunID(row.RunID),
		StepIndex:        int(row.StepIndex),
		Name:             row.Name,
		AgentID:          ptrStr(row.AgentID),
		Status:           workflow.StepStatus(row.Status),
		Input:            input,
		Output:           output,
		RequiresApproval: row.RequiresApproval,
		Timeout:          time.Duration(ptrInt32(row.TimeoutSeconds)) * time.Second,
		MaxRetries:       int(ptrInt32(row.MaxRetries)),
		RetryCount:       int(ptrInt32(row.RetryCount)),
		Error:            ptrStr(row.Error),
		TokensIn:         int(ptrInt32(row.TokensIn)),
		TokensOut:        int(ptrInt32(row.TokensOut)),
		StartedAt:        pgTimestamptzToTimePtr(row.StartedAt),
		CompletedAt:      pgTimestamptzToTimePtr(row.CompletedAt),
		CreatedAt:        pgTimestamptzToTime(row.CreatedAt),
	}, nil
}

// Utility functions

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func int32Ptr(i int32) *int32 {
	return &i
}

func ptrInt32(i *int32) int32 {
	if i == nil {
		return 0
	}
	return *i
}

func timeToPgTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func timeToPgTimestamptzValue(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func pgTimestamptzToTimePtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

func pgTimestamptzToTime(t pgtype.Timestamptz) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time
}

// Ensure WorkflowRepository implements workflow.Repository.
var _ workflow.Repository = (*WorkflowRepository)(nil)
