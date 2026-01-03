package orchestrator

import (
	"context"
	"fmt"
	"time"

	"github.com/felixgeelhaar/bolt"
	"github.com/felixgeelhaar/bridge/internal/domain/agents"
	"github.com/felixgeelhaar/bridge/internal/domain/governance"
	"github.com/felixgeelhaar/bridge/internal/domain/workflow"
	"github.com/felixgeelhaar/bridge/internal/infrastructure/llm"
	"github.com/felixgeelhaar/bridge/pkg/types"
)

// StepResult contains the result of step execution.
type StepResult struct {
	Output   map[string]any
	Tokens   workflow.TokenUsage
	Duration time.Duration
}

// Executor executes individual workflow steps.
type Executor struct {
	logger        *bolt.Logger
	agentRunner   agents.Runner
	agentRegistry *agents.AgentRegistry
	auditService  *governance.AuditService
}

// NewExecutor creates a new step executor.
func NewExecutor(
	logger *bolt.Logger,
	agentRunner agents.Runner,
	agentRegistry *agents.AgentRegistry,
	auditService *governance.AuditService,
) *Executor {
	return &Executor{
		logger:        logger,
		agentRunner:   agentRunner,
		agentRegistry: agentRegistry,
		auditService:  auditService,
	}
}

// ExecuteStep executes a single workflow step.
func (e *Executor) ExecuteStep(ctx context.Context, run *workflow.WorkflowRun, step *workflow.StepRun) (*StepResult, error) {
	logger := e.logger.With().
		Str("run_id", run.ID.String()).
		Str("step_id", step.ID.String()).
		Str("step_name", step.Name).
		Logger()

	// Get agent
	agent, ok := e.agentRegistry.Get(step.AgentID)
	if !ok {
		return nil, fmt.Errorf("%w: %s", types.ErrAgentNotFound, step.AgentID)
	}

	logger.Debug().
		Str("agent", agent.Name).
		Str("model", agent.Model).
		Msg("Executing step with agent")

	// Build input from run context
	input := e.buildStepInput(run, step)

	// Start step
	step.Start(input)

	// Create timeout context
	stepCtx := ctx
	if step.Timeout > 0 {
		var cancel context.CancelFunc
		stepCtx, cancel = context.WithTimeout(ctx, step.Timeout)
		defer cancel()
	}

	// Build messages for agent
	messages := e.buildMessages(run, step, input)

	// Execute agent
	response, err := e.agentRunner.Execute(stepCtx, agent, messages)
	if err != nil {
		return nil, err
	}

	// Log to audit
	e.auditService.LogAgentCalled(
		ctx,
		run.ID.String(),
		step.ID.String(),
		agent.Name,
		response.Model,
		response.TokensIn,
		response.TokensOut,
	)

	// Build output
	output := e.buildOutput(response)

	return &StepResult{
		Output: output,
		Tokens: workflow.TokenUsage{
			Input:  response.TokensIn,
			Output: response.TokensOut,
			Total:  response.TokensIn + response.TokensOut,
		},
		Duration: response.Duration,
	}, nil
}

func (e *Executor) buildStepInput(run *workflow.WorkflowRun, step *workflow.StepRun) map[string]any {
	input := make(map[string]any)

	// Copy step input
	if step.Input != nil {
		for k, v := range step.Input {
			input[k] = v
		}
	}

	// Add trigger data
	if run.TriggerData != nil {
		input["trigger"] = run.TriggerData
	}

	// Add context from previous steps
	if run.Context != nil {
		input["context"] = run.Context
	}

	return input
}

func (e *Executor) buildMessages(run *workflow.WorkflowRun, step *workflow.StepRun, input map[string]any) []llm.Message {
	messages := make([]llm.Message, 0)

	// Build user message with step context
	userContent := fmt.Sprintf("Execute step: %s\n\nInput:\n%v", step.Name, formatInput(input))

	messages = append(messages, llm.Message{
		Role:    llm.RoleUser,
		Content: userContent,
	})

	return messages
}

func (e *Executor) buildOutput(response *agents.AgentResponse) map[string]any {
	output := map[string]any{
		"content":       response.Content,
		"tokens_in":     response.TokensIn,
		"tokens_out":    response.TokensOut,
		"duration_ms":   response.Duration.Milliseconds(),
		"model":         response.Model,
		"finish_reason": string(response.FinishReason),
	}

	if len(response.ToolCalls) > 0 {
		toolCalls := make([]map[string]any, len(response.ToolCalls))
		for i, tc := range response.ToolCalls {
			toolCalls[i] = map[string]any{
				"id":        tc.ID,
				"name":      tc.Name,
				"arguments": tc.Arguments,
			}
		}
		output["tool_calls"] = toolCalls
	}

	return output
}

func formatInput(input map[string]any) string {
	if input == nil {
		return "{}"
	}

	// Simple formatting for now
	result := "{\n"
	for k, v := range input {
		result += fmt.Sprintf("  %s: %v\n", k, v)
	}
	result += "}"
	return result
}
