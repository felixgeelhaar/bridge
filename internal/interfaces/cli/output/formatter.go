package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/felixgeelhaar/bridge/internal/domain/workflow"
)

// Format represents the output format.
type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

// Formatter handles output formatting.
type Formatter struct {
	format Format
	writer io.Writer
}

// NewFormatter creates a new formatter.
func NewFormatter(format string) *Formatter {
	f := FormatText
	if format == "json" {
		f = FormatJSON
	}
	return &Formatter{
		format: f,
		writer: os.Stdout,
	}
}

// Success prints a success message.
func (f *Formatter) Success(message string) {
	if f.format == FormatJSON {
		f.printJSON(map[string]any{
			"status":  "success",
			"message": message,
		})
		return
	}
	_, _ = fmt.Fprintf(f.writer, "✓ %s\n", message)
}

// Error prints an error message.
func (f *Formatter) Error(message string) {
	if f.format == FormatJSON {
		f.printJSON(map[string]any{
			"status":  "error",
			"message": message,
		})
		return
	}
	_, _ = fmt.Fprintf(f.writer, "✗ %s\n", message)
}

// Info prints an info message.
func (f *Formatter) Info(message string) {
	if f.format == FormatJSON {
		f.printJSON(map[string]any{
			"status":  "info",
			"message": message,
		})
		return
	}
	_, _ = fmt.Fprintf(f.writer, "ℹ %s\n", message)
}

// Warning prints a warning message.
func (f *Formatter) Warning(message string) {
	if f.format == FormatJSON {
		f.printJSON(map[string]any{
			"status":  "warning",
			"message": message,
		})
		return
	}
	fmt.Fprintf(f.writer, "⚠ %s\n", message)
}

// WorkflowDefinition prints workflow definition details.
func (f *Formatter) WorkflowDefinition(def *workflow.WorkflowDefinition) {
	if f.format == FormatJSON {
		f.printJSON(map[string]any{
			"id":          def.ID.String(),
			"name":        def.Name,
			"version":     def.Version,
			"description": def.Description,
			"steps":       len(def.Steps),
			"triggers":    len(def.Triggers),
		})
		return
	}

	fmt.Fprintf(f.writer, "Workflow: %s\n", def.Name)
	fmt.Fprintf(f.writer, "  Version:     %s\n", def.Version)
	fmt.Fprintf(f.writer, "  Description: %s\n", def.Description)
	fmt.Fprintf(f.writer, "  Steps:       %d\n", len(def.Steps))
	fmt.Fprintf(f.writer, "  Triggers:    %d\n", len(def.Triggers))
}

// WorkflowRun prints workflow run details.
func (f *Formatter) WorkflowRun(run *workflow.WorkflowRun) {
	if f.format == FormatJSON {
		data := map[string]any{
			"id":            run.ID.String(),
			"workflow_id":   run.WorkflowID.String(),
			"workflow_name": run.WorkflowName,
			"status":        string(run.Status),
			"triggered_by":  run.TriggeredBy,
			"created_at":    run.CreatedAt.Format(time.RFC3339),
		}
		if run.StartedAt != nil {
			data["started_at"] = run.StartedAt.Format(time.RFC3339)
		}
		if run.CompletedAt != nil {
			data["completed_at"] = run.CompletedAt.Format(time.RFC3339)
			data["duration_ms"] = run.Duration().Milliseconds()
		}
		if run.Error != "" {
			data["error"] = run.Error
		}
		f.printJSON(data)
		return
	}

	fmt.Fprintf(f.writer, "Run: %s\n", run.ID.String())
	fmt.Fprintf(f.writer, "  Workflow:     %s\n", run.WorkflowName)
	fmt.Fprintf(f.writer, "  Status:       %s\n", f.statusIcon(run.Status))
	fmt.Fprintf(f.writer, "  Triggered by: %s\n", run.TriggeredBy)
	fmt.Fprintf(f.writer, "  Created:      %s\n", run.CreatedAt.Format(time.RFC3339))

	if run.StartedAt != nil {
		fmt.Fprintf(f.writer, "  Started:      %s\n", run.StartedAt.Format(time.RFC3339))
	}
	if run.CompletedAt != nil {
		fmt.Fprintf(f.writer, "  Completed:    %s\n", run.CompletedAt.Format(time.RFC3339))
		fmt.Fprintf(f.writer, "  Duration:     %s\n", run.Duration())
	}
	if run.Error != "" {
		fmt.Fprintf(f.writer, "  Error:        %s\n", run.Error)
	}

	// Print steps
	if len(run.Steps) > 0 {
		fmt.Fprintf(f.writer, "\n  Steps:\n")
		for _, step := range run.Steps {
			fmt.Fprintf(f.writer, "    %s %s (%s)\n",
				f.stepStatusIcon(step.Status),
				step.Name,
				step.Status,
			)
		}
	}
}

// RunList prints a list of workflow runs.
func (f *Formatter) RunList(runs []*workflow.WorkflowRun) {
	if f.format == FormatJSON {
		list := make([]map[string]any, len(runs))
		for i, run := range runs {
			list[i] = map[string]any{
				"id":            run.ID.String(),
				"workflow_name": run.WorkflowName,
				"status":        string(run.Status),
				"created_at":    run.CreatedAt.Format(time.RFC3339),
			}
		}
		f.printJSON(map[string]any{"runs": list})
		return
	}

	if len(runs) == 0 {
		_, _ = fmt.Fprintln(f.writer, "No active runs")
		return
	}

	w := tabwriter.NewWriter(f.writer, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tWORKFLOW\tSTATUS\tCREATED")
	for _, run := range runs {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			run.ID.String()[:8],
			run.WorkflowName,
			run.Status,
			run.CreatedAt.Format("2006-01-02 15:04:05"),
		)
	}
	_ = w.Flush()
}

// ValidationResult prints validation results.
func (f *Formatter) ValidationResult(valid bool, errors []string, warnings []string) {
	if f.format == FormatJSON {
		f.printJSON(map[string]any{
			"valid":    valid,
			"errors":   errors,
			"warnings": warnings,
		})
		return
	}

	if valid {
		_, _ = fmt.Fprintln(f.writer, "✓ Workflow is valid")
	} else {
		_, _ = fmt.Fprintln(f.writer, "✗ Workflow validation failed")
	}

	if len(errors) > 0 {
		_, _ = fmt.Fprintln(f.writer, "\nErrors:")
		for _, err := range errors {
			_, _ = fmt.Fprintf(f.writer, "  ✗ %s\n", err)
		}
	}

	if len(warnings) > 0 {
		_, _ = fmt.Fprintln(f.writer, "\nWarnings:")
		for _, warn := range warnings {
			_, _ = fmt.Fprintf(f.writer, "  ⚠ %s\n", warn)
		}
	}
}

// ApprovalStatus prints approval status.
func (f *Formatter) ApprovalStatus(runID string, status string, approvedBy string) {
	if f.format == FormatJSON {
		f.printJSON(map[string]any{
			"run_id":      runID,
			"status":      status,
			"approved_by": approvedBy,
		})
		return
	}

	switch status {
	case "approved":
		fmt.Fprintf(f.writer, "✓ Run %s approved by %s\n", runID[:8], approvedBy)
	case "rejected":
		fmt.Fprintf(f.writer, "✗ Run %s rejected by %s\n", runID[:8], approvedBy)
	default:
		fmt.Fprintf(f.writer, "ℹ Run %s: %s\n", runID[:8], status)
	}
}

func (f *Formatter) statusIcon(status workflow.RunStatus) string {
	switch status {
	case workflow.RunStatusPending:
		return "⏳ pending"
	case workflow.RunStatusExecuting:
		return "▶ executing"
	case workflow.RunStatusCompleted:
		return "✓ completed"
	case workflow.RunStatusFailed:
		return "✗ failed"
	case workflow.RunStatusCancelled:
		return "⊘ cancelled"
	case workflow.RunStatusAwaitingApproval:
		return "⏸ awaiting approval"
	default:
		return string(status)
	}
}

func (f *Formatter) stepStatusIcon(status workflow.StepStatus) string {
	switch status {
	case workflow.StepStatusPending:
		return "○"
	case workflow.StepStatusRunning:
		return "▶"
	case workflow.StepStatusCompleted:
		return "✓"
	case workflow.StepStatusFailed:
		return "✗"
	case workflow.StepStatusSkipped:
		return "⊘"
	default:
		return "?"
	}
}

func (f *Formatter) printJSON(data any) {
	enc := json.NewEncoder(f.writer)
	enc.SetIndent("", "  ")
	enc.Encode(data)
}

// Table prints data as a table.
func (f *Formatter) Table(headers []string, rows [][]string) {
	if f.format == FormatJSON {
		data := make([]map[string]string, len(rows))
		for i, row := range rows {
			data[i] = make(map[string]string)
			for j, cell := range row {
				if j < len(headers) {
					data[i][strings.ToLower(headers[j])] = cell
				}
			}
		}
		f.printJSON(map[string]any{"data": data})
		return
	}

	w := tabwriter.NewWriter(f.writer, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, strings.Join(headers, "\t"))
	for _, row := range rows {
		_, _ = fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	_ = w.Flush()
}
