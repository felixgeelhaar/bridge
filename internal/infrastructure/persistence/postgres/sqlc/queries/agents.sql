-- name: CreateAgent :one
INSERT INTO agents (
    id, name, description, provider, model, system_prompt,
    max_tokens, temperature, capabilities, metadata, active,
    created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
)
RETURNING *;

-- name: GetAgent :one
SELECT * FROM agents
WHERE id = $1;

-- name: GetAgentByName :one
SELECT * FROM agents
WHERE name = $1;

-- name: ListAgents :many
SELECT * FROM agents
ORDER BY name ASC
LIMIT $1 OFFSET $2;

-- name: ListActiveAgents :many
SELECT * FROM agents
WHERE active = TRUE
ORDER BY name ASC;

-- name: UpdateAgent :one
UPDATE agents
SET
    name = $2,
    description = $3,
    provider = $4,
    model = $5,
    system_prompt = $6,
    max_tokens = $7,
    temperature = $8,
    capabilities = $9,
    metadata = $10,
    active = $11,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteAgent :exec
DELETE FROM agents
WHERE id = $1;

-- name: CountAgents :one
SELECT COUNT(*) FROM agents;
