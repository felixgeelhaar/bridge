package memory

import (
	"context"
	"sync"

	"github.com/felixgeelhaar/bridge/internal/domain/workflow"
	"github.com/felixgeelhaar/bridge/pkg/types"
)

// WorkflowRepository is an in-memory implementation of workflow.Repository.
type WorkflowRepository struct {
	mu          sync.RWMutex
	definitions map[types.WorkflowID]*workflow.WorkflowDefinition
	nameIndex   map[string]types.WorkflowID
	runs        map[types.RunID]*workflow.WorkflowRun
	steps       map[types.StepID]*workflow.StepRun
}

// NewWorkflowRepository creates a new in-memory workflow repository.
func NewWorkflowRepository() *WorkflowRepository {
	return &WorkflowRepository{
		definitions: make(map[types.WorkflowID]*workflow.WorkflowDefinition),
		nameIndex:   make(map[string]types.WorkflowID),
		runs:        make(map[types.RunID]*workflow.WorkflowRun),
		steps:       make(map[types.StepID]*workflow.StepRun),
	}
}

// CreateDefinition creates a new workflow definition.
func (r *WorkflowRepository) CreateDefinition(ctx context.Context, def *workflow.WorkflowDefinition) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.nameIndex[def.Name]; exists {
		return types.ErrWorkflowAlreadyExists
	}

	r.definitions[def.ID] = def
	r.nameIndex[def.Name] = def.ID
	return nil
}

// GetDefinition retrieves a workflow definition by ID.
func (r *WorkflowRepository) GetDefinition(ctx context.Context, id types.WorkflowID) (*workflow.WorkflowDefinition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	def, ok := r.definitions[id]
	if !ok {
		return nil, types.ErrWorkflowNotFound
	}
	return def, nil
}

// GetDefinitionByName retrieves a workflow definition by name.
func (r *WorkflowRepository) GetDefinitionByName(ctx context.Context, name string) (*workflow.WorkflowDefinition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, ok := r.nameIndex[name]
	if !ok {
		return nil, types.ErrWorkflowNotFound
	}
	return r.definitions[id], nil
}

// ListDefinitions lists workflow definitions with pagination.
func (r *WorkflowRepository) ListDefinitions(ctx context.Context, limit, offset int) ([]*workflow.WorkflowDefinition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	defs := make([]*workflow.WorkflowDefinition, 0, len(r.definitions))
	for _, def := range r.definitions {
		defs = append(defs, def)
	}

	// Apply pagination
	if offset >= len(defs) {
		return []*workflow.WorkflowDefinition{}, nil
	}

	end := offset + limit
	if end > len(defs) {
		end = len(defs)
	}

	return defs[offset:end], nil
}

// UpdateDefinition updates a workflow definition.
func (r *WorkflowRepository) UpdateDefinition(ctx context.Context, def *workflow.WorkflowDefinition) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.definitions[def.ID]; !ok {
		return types.ErrWorkflowNotFound
	}

	r.definitions[def.ID] = def
	return nil
}

// DeleteDefinition deletes a workflow definition.
func (r *WorkflowRepository) DeleteDefinition(ctx context.Context, id types.WorkflowID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	def, ok := r.definitions[id]
	if !ok {
		return types.ErrWorkflowNotFound
	}

	delete(r.definitions, id)
	delete(r.nameIndex, def.Name)
	return nil
}

// CreateRun creates a new workflow run.
func (r *WorkflowRepository) CreateRun(ctx context.Context, run *workflow.WorkflowRun) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.runs[run.ID] = run

	// Index all steps
	for _, step := range run.Steps {
		r.steps[step.ID] = step
	}

	return nil
}

// GetRun retrieves a workflow run by ID.
func (r *WorkflowRepository) GetRun(ctx context.Context, id types.RunID) (*workflow.WorkflowRun, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	run, ok := r.runs[id]
	if !ok {
		return nil, types.ErrRunNotFound
	}
	return run, nil
}

// ListRuns lists workflow runs for a given workflow.
func (r *WorkflowRepository) ListRuns(ctx context.Context, workflowID types.WorkflowID, limit, offset int) ([]*workflow.WorkflowRun, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	runs := make([]*workflow.WorkflowRun, 0)
	for _, run := range r.runs {
		if run.WorkflowID == workflowID {
			runs = append(runs, run)
		}
	}

	// Apply pagination
	if offset >= len(runs) {
		return []*workflow.WorkflowRun{}, nil
	}

	end := offset + limit
	if end > len(runs) {
		end = len(runs)
	}

	return runs[offset:end], nil
}

// ListActiveRuns lists all active workflow runs.
func (r *WorkflowRepository) ListActiveRuns(ctx context.Context) ([]*workflow.WorkflowRun, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	runs := make([]*workflow.WorkflowRun, 0)
	for _, run := range r.runs {
		if !run.Status.IsTerminal() {
			runs = append(runs, run)
		}
	}

	return runs, nil
}

// UpdateRun updates a workflow run.
func (r *WorkflowRepository) UpdateRun(ctx context.Context, run *workflow.WorkflowRun) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.runs[run.ID]; !ok {
		return types.ErrRunNotFound
	}

	r.runs[run.ID] = run
	return nil
}

// GetStep retrieves a step run by ID.
func (r *WorkflowRepository) GetStep(ctx context.Context, id types.StepID) (*workflow.StepRun, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	step, ok := r.steps[id]
	if !ok {
		return nil, types.ErrStepNotFound
	}
	return step, nil
}

// UpdateStep updates a step run.
func (r *WorkflowRepository) UpdateStep(ctx context.Context, step *workflow.StepRun) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.steps[step.ID]; !ok {
		return types.ErrStepNotFound
	}

	r.steps[step.ID] = step
	return nil
}

// Ensure WorkflowRepository implements workflow.Repository.
var _ workflow.Repository = (*WorkflowRepository)(nil)
