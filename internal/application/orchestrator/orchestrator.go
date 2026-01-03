package orchestrator

import (
	"context"
	"fmt"

	"github.com/felixgeelhaar/bolt"
	"github.com/felixgeelhaar/bridge/internal/domain/agents"
	"github.com/felixgeelhaar/bridge/internal/domain/governance"
	"github.com/felixgeelhaar/bridge/internal/domain/workflow"
	"github.com/felixgeelhaar/bridge/internal/infrastructure/llm"
	"github.com/felixgeelhaar/bridge/pkg/config"
	"github.com/felixgeelhaar/bridge/pkg/types"
)

// Orchestrator coordinates workflow execution.
type Orchestrator struct {
	logger          *bolt.Logger
	workflowService *workflow.Service
	policyEvaluator governance.Evaluator
	auditService    *governance.AuditService
	agentRunner     agents.Runner
	agentRegistry   *agents.AgentRegistry
	stateMachine    *workflow.RunStateMachine
}

// Config contains orchestrator configuration.
type Config struct {
	Logger          *bolt.Logger
	WorkflowRepo    workflow.Repository
	EventPublisher  workflow.EventPublisher
	PolicyEvaluator governance.Evaluator
	AuditLogger     governance.AuditLogger
	LLMRegistry     *llm.Registry
	AgentRegistry   *agents.AgentRegistry
}

// New creates a new orchestrator.
func New(cfg Config) (*Orchestrator, error) {
	sm, err := workflow.NewRunStateMachine()
	if err != nil {
		return nil, fmt.Errorf("failed to create state machine: %w", err)
	}

	return &Orchestrator{
		logger:          cfg.Logger,
		workflowService: workflow.NewService(cfg.WorkflowRepo, cfg.EventPublisher),
		policyEvaluator: cfg.PolicyEvaluator,
		auditService:    governance.NewAuditService(cfg.AuditLogger),
		agentRunner:     agents.NewRunner(cfg.Logger, cfg.LLMRegistry),
		agentRegistry:   cfg.AgentRegistry,
		stateMachine:    sm,
	}, nil
}

// CreateWorkflow creates a new workflow definition from config.
func (o *Orchestrator) CreateWorkflow(ctx context.Context, cfg *config.WorkflowConfig) (*workflow.WorkflowDefinition, error) {
	def, err := workflow.NewWorkflowDefinition(cfg)
	if err != nil {
		return nil, err
	}

	if err := o.workflowService.CreateWorkflow(ctx, def); err != nil {
		return nil, err
	}

	o.logger.Info().
		Str("workflow_id", def.ID.String()).
		Str("name", def.Name).
		Str("version", def.Version).
		Msg("Workflow created")

	return def, nil
}

// CreateRun creates a new workflow run.
func (o *Orchestrator) CreateRun(ctx context.Context, def *workflow.WorkflowDefinition, triggeredBy string, triggerData map[string]any) (*workflow.WorkflowRun, error) {
	run, err := o.workflowService.StartRun(ctx, def, triggeredBy, triggerData)
	if err != nil {
		return nil, err
	}

	o.auditService.LogWorkflowStarted(ctx, def.ID.String(), run.ID.String(), triggeredBy)

	o.logger.Info().
		Str("run_id", run.ID.String()).
		Str("workflow", def.Name).
		Str("triggered_by", triggeredBy).
		Msg("Workflow run created")

	return run, nil
}

// ExecuteWorkflow executes a workflow run.
func (o *Orchestrator) ExecuteWorkflow(ctx context.Context, run *workflow.WorkflowRun) error {
	logger := o.logger.With().
		Str("run_id", run.ID.String()).
		Str("workflow", run.WorkflowName).
		Logger()

	logger.Info().Msg("Starting workflow execution")

	// Initialize state machine
	runCtx := workflow.RunContext{Run: run}
	interp, err := o.stateMachine.Start(runCtx)
	if err != nil {
		return fmt.Errorf("failed to start state machine: %w", err)
	}

	// Start execution
	run.Start()
	if err := o.workflowService.UpdateRun(ctx, run); err != nil {
		return err
	}

	// Send START event
	if err := o.stateMachine.Send(interp, workflow.EventStart); err != nil {
		return err
	}

	// Policy check
	policyResult, err := o.evaluatePolicy(ctx, run)
	if err != nil {
		run.Fail(fmt.Sprintf("policy evaluation failed: %v", err))
		o.workflowService.UpdateRun(ctx, run)
		return err
	}

	if !policyResult.Allowed {
		run.Fail("policy violation: " + formatViolations(policyResult.Violations))
		o.workflowService.UpdateRun(ctx, run)
		o.auditService.LogWorkflowFailed(ctx, run.WorkflowID.String(), run.ID.String(), run.Error)
		return types.ErrPolicyViolation
	}

	// Send policy pass event
	if err := o.stateMachine.Send(interp, workflow.EventPolicyPass); err != nil {
		return err
	}

	// Check if approval is required
	if policyResult.RequiresApproval {
		if err := o.stateMachine.Send(interp, workflow.EventApprovalRequired); err != nil {
			return err
		}
		run.AwaitApproval()
		o.workflowService.UpdateRun(ctx, run)
		logger.Info().Msg("Workflow awaiting approval")
		return types.ErrApprovalRequired
	}

	// No approval needed, proceed to execution
	if err := o.stateMachine.Send(interp, workflow.EventNoApproval); err != nil {
		return err
	}

	// Execute steps
	return o.executeSteps(ctx, run, interp, logger)
}

// ResumeWorkflow resumes a workflow after approval.
func (o *Orchestrator) ResumeWorkflow(ctx context.Context, run *workflow.WorkflowRun) error {
	if run.Status != workflow.RunStatusAwaitingApproval {
		return fmt.Errorf("workflow is not awaiting approval")
	}

	logger := o.logger.With().
		Str("run_id", run.ID.String()).
		Str("workflow", run.WorkflowName).
		Logger()

	logger.Info().Msg("Resuming workflow after approval")

	// Initialize state machine at awaiting_approval state
	runCtx := workflow.RunContext{Run: run}
	interp, err := o.stateMachine.Start(runCtx)
	if err != nil {
		return err
	}

	// Send approved event
	if err := o.stateMachine.Send(interp, workflow.EventApproved); err != nil {
		return err
	}

	run.Approve()
	o.workflowService.UpdateRun(ctx, run)

	return o.executeSteps(ctx, run, interp, logger)
}

func (o *Orchestrator) executeSteps(ctx context.Context, run *workflow.WorkflowRun, interp *workflow.Interpreter, logger *bolt.Logger) error {
	executor := NewExecutor(o.logger, o.agentRunner, o.agentRegistry, o.auditService)

	for run.HasMoreSteps() {
		step := run.CurrentStep()

		logger.Info().
			Str("step", step.Name).
			Int("index", step.StepIndex).
			Msg("Executing step")

		// Execute step
		result, err := executor.ExecuteStep(ctx, run, step)
		if err != nil {
			step.Fail(err.Error())
			o.workflowService.UpdateStep(ctx, step)

			// Check if can retry
			if step.CanRetry() {
				step.IncrementRetry()
				o.workflowService.UpdateStep(ctx, step)
				logger.Warn().
					Str("step", step.Name).
					Int("retry", step.RetryCount).
					Err(err).
					Msg("Step failed, retrying")
				continue
			}

			// No retry, fail workflow
			run.Fail(fmt.Sprintf("step %s failed: %v", step.Name, err))
			o.workflowService.UpdateRun(ctx, run)
			o.auditService.LogWorkflowFailed(ctx, run.WorkflowID.String(), run.ID.String(), run.Error)
			return err
		}

		// Step completed
		step.Complete(result.Output, result.Tokens.Input, result.Tokens.Output)
		o.workflowService.UpdateStep(ctx, step)

		// Store output in context
		run.SetContext(fmt.Sprintf("steps.%s.output", step.Name), result.Output)
		o.workflowService.UpdateRun(ctx, run)

		logger.Info().
			Str("step", step.Name).
			Dur("duration", result.Duration).
			Int("tokens_in", result.Tokens.Input).
			Int("tokens_out", result.Tokens.Output).
			Msg("Step completed")

		// Advance to next step
		run.AdvanceStep()
	}

	// All steps completed
	run.Complete()
	o.workflowService.UpdateRun(ctx, run)
	o.auditService.LogWorkflowCompleted(ctx, run.WorkflowID.String(), run.ID.String(), run.Duration())

	logger.Info().
		Dur("duration", run.Duration()).
		Msg("Workflow completed successfully")

	return nil
}

func (o *Orchestrator) evaluatePolicy(ctx context.Context, run *workflow.WorkflowRun) (*governance.PolicyResult, error) {
	input := &governance.PolicyInput{
		WorkflowID:   run.WorkflowID.String(),
		WorkflowName: run.WorkflowName,
		RunID:        run.ID.String(),
		Context:      run.Context,
	}

	result, err := o.policyEvaluator.EvaluateAll(ctx, input)
	if err != nil {
		return nil, err
	}

	o.auditService.LogPolicyEvaluated(ctx, run.ID.String(), "all", result.Allowed)

	for _, v := range result.Violations {
		o.auditService.LogPolicyViolation(ctx, run.ID.String(), v.Rule, v.Message)
	}

	return result, nil
}

func formatViolations(violations []governance.Violation) string {
	if len(violations) == 0 {
		return "unknown violation"
	}

	msgs := make([]string, len(violations))
	for i, v := range violations {
		msgs[i] = v.Message
	}
	return fmt.Sprintf("%v", msgs)
}

// GetRun retrieves a workflow run.
func (o *Orchestrator) GetRun(ctx context.Context, id types.RunID) (*workflow.WorkflowRun, error) {
	return o.workflowService.GetRun(ctx, id)
}

// ListActiveRuns returns all active workflow runs.
func (o *Orchestrator) ListActiveRuns(ctx context.Context) ([]*workflow.WorkflowRun, error) {
	return o.workflowService.ListActiveRuns(ctx)
}

// Interpreter type alias is defined in the statemachine package
