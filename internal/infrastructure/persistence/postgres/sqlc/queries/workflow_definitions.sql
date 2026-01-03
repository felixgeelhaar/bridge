-- name: CreateWorkflowDefinition :one
INSERT INTO workflow_definitions (
    id, name, version, description, config, checksum, metadata, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING *;

-- name: GetWorkflowDefinition :one
SELECT * FROM workflow_definitions
WHERE id = $1;

-- name: GetWorkflowDefinitionByName :one
SELECT * FROM workflow_definitions
WHERE name = $1;

-- name: ListWorkflowDefinitions :many
SELECT * FROM workflow_definitions
ORDER BY updated_at DESC
LIMIT $1 OFFSET $2;

-- name: CountWorkflowDefinitions :one
SELECT COUNT(*) FROM workflow_definitions;

-- name: UpdateWorkflowDefinition :one
UPDATE workflow_definitions
SET
    name = $2,
    version = $3,
    description = $4,
    config = $5,
    checksum = $6,
    metadata = $7,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteWorkflowDefinition :exec
DELETE FROM workflow_definitions
WHERE id = $1;
