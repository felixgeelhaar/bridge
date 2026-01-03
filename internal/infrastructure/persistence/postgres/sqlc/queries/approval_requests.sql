-- name: CreateApprovalRequest :one
INSERT INTO approval_requests (
    id, run_id, step_name, status, requested_by,
    approved_by, rejected_by, reason, expires_at, decided_at, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
)
RETURNING *;

-- name: GetApprovalRequest :one
SELECT * FROM approval_requests
WHERE id = $1;

-- name: GetApprovalRequestByRunID :one
SELECT * FROM approval_requests
WHERE run_id = $1 AND status = 'pending'
ORDER BY created_at DESC
LIMIT 1;

-- name: ListApprovalRequestsByRunID :many
SELECT * FROM approval_requests
WHERE run_id = $1
ORDER BY created_at DESC;

-- name: ListPendingApprovalRequests :many
SELECT * FROM approval_requests
WHERE status = 'pending'
  AND (expires_at IS NULL OR expires_at > NOW())
ORDER BY created_at ASC;

-- name: UpdateApprovalRequest :one
UPDATE approval_requests
SET
    status = $2,
    approved_by = $3,
    rejected_by = $4,
    reason = $5,
    decided_at = $6
WHERE id = $1
RETURNING *;

-- name: DeleteApprovalRequest :exec
DELETE FROM approval_requests
WHERE id = $1;

-- name: ExpireApprovalRequests :exec
UPDATE approval_requests
SET status = 'expired', decided_at = NOW()
WHERE status = 'pending' AND expires_at < NOW();
