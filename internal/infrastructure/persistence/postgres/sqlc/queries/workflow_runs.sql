-- name: CreateWorkflowRun :one
INSERT INTO workflow_runs (
    id, workflow_id, workflow_name, workflow_version, status,
    current_step_index, context, triggered_by, trigger_data,
    error, started_at, completed_at, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
)
RETURNING *;

-- name: GetWorkflowRun :one
SELECT * FROM workflow_runs
WHERE id = $1;

-- name: ListWorkflowRuns :many
SELECT * FROM workflow_runs
WHERE workflow_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListActiveWorkflowRuns :many
SELECT * FROM workflow_runs
WHERE status NOT IN ('completed', 'failed', 'cancelled')
ORDER BY created_at DESC;

-- name: UpdateWorkflowRun :one
UPDATE workflow_runs
SET
    status = $2,
    current_step_index = $3,
    context = $4,
    error = $5,
    started_at = $6,
    completed_at = $7,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: CountWorkflowRuns :one
SELECT COUNT(*) FROM workflow_runs
WHERE workflow_id = $1;

-- name: CountActiveWorkflowRuns :one
SELECT COUNT(*) FROM workflow_runs
WHERE status NOT IN ('completed', 'failed', 'cancelled');
