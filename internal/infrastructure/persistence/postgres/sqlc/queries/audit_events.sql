-- name: CreateAuditEvent :one
INSERT INTO audit_events (
    id, type, actor, resource_type, resource_id, action, details, timestamp
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: GetAuditEvent :one
SELECT * FROM audit_events
WHERE id = $1;

-- name: ListAuditEvents :many
SELECT * FROM audit_events
ORDER BY timestamp DESC
LIMIT $1 OFFSET $2;

-- name: ListAuditEventsByResource :many
SELECT * FROM audit_events
WHERE resource_type = $1 AND resource_id = $2
ORDER BY timestamp DESC
LIMIT $3 OFFSET $4;

-- name: ListAuditEventsByActor :many
SELECT * FROM audit_events
WHERE actor = $1
ORDER BY timestamp DESC
LIMIT $2 OFFSET $3;

-- name: ListAuditEventsByType :many
SELECT * FROM audit_events
WHERE type = $1
ORDER BY timestamp DESC
LIMIT $2 OFFSET $3;

-- name: CountAuditEvents :one
SELECT COUNT(*) FROM audit_events;

-- name: DeleteOldAuditEvents :exec
DELETE FROM audit_events
WHERE timestamp < $1;
