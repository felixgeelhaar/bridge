-- name: CreateStepRun :one
INSERT INTO step_runs (
    id, run_id, step_index, name, agent_id, status,
    input, output, requires_approval, timeout_seconds,
    max_retries, retry_count, error, tokens_in, tokens_out,
    step_order, started_at, completed_at, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
    $11, $12, $13, $14, $15, $16, $17, $18, $19
)
RETURNING *;

-- name: GetStepRun :one
SELECT * FROM step_runs
WHERE id = $1;

-- name: ListStepRunsByRunID :many
SELECT * FROM step_runs
WHERE run_id = $1
ORDER BY step_index ASC;

-- name: UpdateStepRun :one
UPDATE step_runs
SET
    status = $2,
    input = $3,
    output = $4,
    retry_count = $5,
    error = $6,
    tokens_in = $7,
    tokens_out = $8,
    started_at = $9,
    completed_at = $10
WHERE id = $1
RETURNING *;

-- name: DeleteStepRunsByRunID :exec
DELETE FROM step_runs
WHERE run_id = $1;
