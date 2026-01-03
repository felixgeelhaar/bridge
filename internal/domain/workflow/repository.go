package workflow

import (
	"context"

	"github.com/felixgeelhaar/bridge/pkg/types"
)

// Repository defines the interface for workflow persistence.
type Repository interface {
	// Definition operations
	CreateDefinition(ctx context.Context, def *WorkflowDefinition) error
	GetDefinition(ctx context.Context, id types.WorkflowID) (*WorkflowDefinition, error)
	GetDefinitionByName(ctx context.Context, name string) (*WorkflowDefinition, error)
	ListDefinitions(ctx context.Context, limit, offset int) ([]*WorkflowDefinition, error)
	UpdateDefinition(ctx context.Context, def *WorkflowDefinition) error
	DeleteDefinition(ctx context.Context, id types.WorkflowID) error

	// Run operations
	CreateRun(ctx context.Context, run *WorkflowRun) error
	GetRun(ctx context.Context, id types.RunID) (*WorkflowRun, error)
	ListRuns(ctx context.Context, workflowID types.WorkflowID, limit, offset int) ([]*WorkflowRun, error)
	ListActiveRuns(ctx context.Context) ([]*WorkflowRun, error)
	UpdateRun(ctx context.Context, run *WorkflowRun) error

	// Step operations
	GetStep(ctx context.Context, id types.StepID) (*StepRun, error)
	UpdateStep(ctx context.Context, step *StepRun) error
}

// EventPublisher publishes domain events.
type EventPublisher interface {
	Publish(ctx context.Context, event Event) error
}

// Service provides workflow domain operations.
type Service struct {
	repo      Repository
	publisher EventPublisher
}

// NewService creates a new workflow service.
func NewService(repo Repository, publisher EventPublisher) *Service {
	return &Service{
		repo:      repo,
		publisher: publisher,
	}
}

// CreateWorkflow creates a new workflow definition.
func (s *Service) CreateWorkflow(ctx context.Context, def *WorkflowDefinition) error {
	if err := s.repo.CreateDefinition(ctx, def); err != nil {
		return err
	}

	if s.publisher != nil {
		return s.publisher.Publish(ctx, NewWorkflowCreatedEvent(def))
	}
	return nil
}

// GetWorkflow retrieves a workflow definition by ID.
func (s *Service) GetWorkflow(ctx context.Context, id types.WorkflowID) (*WorkflowDefinition, error) {
	return s.repo.GetDefinition(ctx, id)
}

// GetWorkflowByName retrieves a workflow definition by name.
func (s *Service) GetWorkflowByName(ctx context.Context, name string) (*WorkflowDefinition, error) {
	return s.repo.GetDefinitionByName(ctx, name)
}

// StartRun creates and starts a new workflow run.
func (s *Service) StartRun(ctx context.Context, def *WorkflowDefinition, triggeredBy string, triggerData map[string]any) (*WorkflowRun, error) {
	run := NewWorkflowRun(def, triggeredBy, triggerData)

	if err := s.repo.CreateRun(ctx, run); err != nil {
		return nil, err
	}

	if s.publisher != nil {
		if err := s.publisher.Publish(ctx, NewRunStartedEvent(run)); err != nil {
			return nil, err
		}
	}

	return run, nil
}

// GetRun retrieves a workflow run by ID.
func (s *Service) GetRun(ctx context.Context, id types.RunID) (*WorkflowRun, error) {
	return s.repo.GetRun(ctx, id)
}

// UpdateRun updates a workflow run.
func (s *Service) UpdateRun(ctx context.Context, run *WorkflowRun) error {
	return s.repo.UpdateRun(ctx, run)
}

// CompleteRun marks a workflow run as completed.
func (s *Service) CompleteRun(ctx context.Context, run *WorkflowRun) error {
	run.Complete()

	if err := s.repo.UpdateRun(ctx, run); err != nil {
		return err
	}

	if s.publisher != nil {
		return s.publisher.Publish(ctx, NewRunCompletedEvent(run))
	}
	return nil
}

// FailRun marks a workflow run as failed.
func (s *Service) FailRun(ctx context.Context, run *WorkflowRun, err string, failedStep string) error {
	run.Fail(err)

	if updateErr := s.repo.UpdateRun(ctx, run); updateErr != nil {
		return updateErr
	}

	if s.publisher != nil {
		return s.publisher.Publish(ctx, NewRunFailedEvent(run, failedStep))
	}
	return nil
}

// ListActiveRuns returns all active workflow runs.
func (s *Service) ListActiveRuns(ctx context.Context) ([]*WorkflowRun, error) {
	return s.repo.ListActiveRuns(ctx)
}

// UpdateStep updates a step run.
func (s *Service) UpdateStep(ctx context.Context, step *StepRun) error {
	return s.repo.UpdateStep(ctx, step)
}
