package workflow

import (
	"context"
	"fmt"

	"github.com/felixgeelhaar/statekit"
)

// RunEvent represents events that can trigger state transitions.
type RunEvent string

const (
	EventStart            RunEvent = "START"
	EventPolicyPass       RunEvent = "POLICY_PASS"
	EventPolicyFail       RunEvent = "POLICY_FAIL"
	EventApprovalRequired RunEvent = "APPROVAL_REQUIRED"
	EventNoApproval       RunEvent = "NO_APPROVAL"
	EventApproved         RunEvent = "APPROVED"
	EventRejected         RunEvent = "REJECTED"
	EventStepComplete     RunEvent = "STEP_COMPLETE"
	EventStepFailed       RunEvent = "STEP_FAILED"
	EventAllDone          RunEvent = "ALL_DONE"
	EventHasNext          RunEvent = "HAS_NEXT"
	EventRetry            RunEvent = "RETRY"
	EventAbort            RunEvent = "ABORT"
	EventCancel           RunEvent = "CANCEL"
	EventTimeout          RunEvent = "TIMEOUT"
)

// RunContext holds the runtime context for workflow state machine.
type RunContext struct {
	Run            *WorkflowRun
	CurrentStep    *StepRun
	PolicyResult   *PolicyEvaluationResult
	ApprovalResult *ApprovalResult
	LastError      error
}

// PolicyEvaluationResult represents the result of policy evaluation.
type PolicyEvaluationResult struct {
	Allowed          bool
	RequiresApproval bool
	Reason           string
	Violations       []string
}

// ApprovalResult represents the result of an approval request.
type ApprovalResult struct {
	Approved   bool
	ApprovedBy string
	Reason     string
}

// RunStateMachine manages workflow run state transitions.
type RunStateMachine struct {
	machine *statekit.MachineConfig[RunContext]
}

// NewRunStateMachine creates a new workflow run state machine.
func NewRunStateMachine() (*RunStateMachine, error) {
	machine, err := statekit.NewMachine[RunContext]("workflow_run").
		WithInitial("pending").
		// Initial state - workflow scheduled but not started
		State("pending").
		On(statekit.EventType(EventStart)).Target("policy_check").
		On(statekit.EventType(EventCancel)).Target("cancelled").
		Done().
		// Policy evaluation before execution
		State("policy_check").
		On(statekit.EventType(EventPolicyPass)).Target("check_approval").
		On(statekit.EventType(EventPolicyFail)).Target("failed").
		Done().
		// Check if approval is required
		State("check_approval").
		On(statekit.EventType(EventApprovalRequired)).Target("awaiting_approval").
		On(statekit.EventType(EventNoApproval)).Target("executing").
		Done().
		// Waiting for human approval
		State("awaiting_approval").
		On(statekit.EventType(EventApproved)).Target("executing").
		On(statekit.EventType(EventRejected)).Target("cancelled").
		On(statekit.EventType(EventTimeout)).Target("failed").
		On(statekit.EventType(EventCancel)).Target("cancelled").
		Done().
		// Main execution state
		State("executing").
		On(statekit.EventType(EventStepComplete)).Target("check_next").
		On(statekit.EventType(EventStepFailed)).Target("step_failed").
		On(statekit.EventType(EventCancel)).Target("cancelled").
		Done().
		// Check if there are more steps
		State("check_next").
		On(statekit.EventType(EventHasNext)).Target("executing").
		On(statekit.EventType(EventAllDone)).Target("completed").
		Done().
		// Handle step failure
		State("step_failed").
		On(statekit.EventType(EventRetry)).Target("executing").
		On(statekit.EventType(EventAbort)).Target("failed").
		Done().
		// Terminal states
		State("completed").Final().Done().
		State("failed").Final().Done().
		State("cancelled").Final().Done().
		Build()

	if err != nil {
		return nil, fmt.Errorf("failed to build state machine: %w", err)
	}

	return &RunStateMachine{machine: machine}, nil
}

// Interpreter is a type alias for the statekit interpreter.
type Interpreter = statekit.Interpreter[RunContext]

// Start initializes and starts the state machine interpreter.
func (sm *RunStateMachine) Start(ctx RunContext) (*Interpreter, error) {
	interp := statekit.NewInterpreter(sm.machine)
	interp.Start()
	return interp, nil
}

// Send sends an event to the interpreter.
func (sm *RunStateMachine) Send(interp *Interpreter, event RunEvent) error {
	interp.Send(statekit.Event{Type: statekit.EventType(event)})
	return nil
}

// CurrentState returns the current state name.
func (sm *RunStateMachine) CurrentState(interp *Interpreter) string {
	return string(interp.State().Value)
}

// IsDone returns true if the state machine is in a final state.
func (sm *RunStateMachine) IsDone(interp *Interpreter) bool {
	return interp.Done()
}

// MapStatusToState maps a RunStatus to the corresponding state machine state.
func MapStatusToState(status RunStatus) string {
	switch status {
	case RunStatusPending:
		return "pending"
	case RunStatusPolicyCheck:
		return "policy_check"
	case RunStatusAwaitingApproval:
		return "awaiting_approval"
	case RunStatusExecuting:
		return "executing"
	case RunStatusCompleted:
		return "completed"
	case RunStatusFailed:
		return "failed"
	case RunStatusCancelled:
		return "cancelled"
	default:
		return "pending"
	}
}

// MapStateToStatus maps a state machine state to RunStatus.
func MapStateToStatus(state string) RunStatus {
	switch state {
	case "pending":
		return RunStatusPending
	case "policy_check":
		return RunStatusPolicyCheck
	case "check_approval":
		return RunStatusPolicyCheck // Intermediate state
	case "awaiting_approval":
		return RunStatusAwaitingApproval
	case "executing", "check_next", "step_failed":
		return RunStatusExecuting
	case "completed":
		return RunStatusCompleted
	case "failed":
		return RunStatusFailed
	case "cancelled":
		return RunStatusCancelled
	default:
		return RunStatusPending
	}
}

// StepStateMachine manages step execution state transitions.
type StepStateMachine struct {
	machine *statekit.MachineConfig[*StepRun]
}

// NewStepStateMachine creates a new step state machine.
func NewStepStateMachine() (*StepStateMachine, error) {
	machine, err := statekit.NewMachine[*StepRun]("step_execution").
		WithInitial("pending").
		State("pending").
		On(statekit.EventType("EXECUTE")).Target("running").
		On(statekit.EventType("SKIP")).Target("skipped").
		Done().
		State("running").
		On(statekit.EventType("SUCCESS")).Target("completed").
		On(statekit.EventType("FAILURE")).Target("failed").
		On(statekit.EventType("TIMEOUT")).Target("failed").
		Done().
		State("completed").Final().Done().
		State("failed").Final().Done().
		State("skipped").Final().Done().
		Build()

	if err != nil {
		return nil, fmt.Errorf("failed to build step state machine: %w", err)
	}

	return &StepStateMachine{machine: machine}, nil
}

// StateMachineObserver observes state machine transitions.
type StateMachineObserver interface {
	OnStateChange(ctx context.Context, runID string, fromState, toState string)
	OnEvent(ctx context.Context, runID string, event string)
}
