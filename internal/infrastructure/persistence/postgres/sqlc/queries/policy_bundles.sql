-- name: CreatePolicyBundle :one
INSERT INTO policy_bundles (
    id, name, version, description, rules, checksum, active,
    created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING *;

-- name: GetPolicyBundle :one
SELECT * FROM policy_bundles
WHERE id = $1;

-- name: GetPolicyBundleByName :one
SELECT * FROM policy_bundles
WHERE name = $1;

-- name: ListPolicyBundles :many
SELECT * FROM policy_bundles
ORDER BY name ASC
LIMIT $1 OFFSET $2;

-- name: ListActivePolicyBundles :many
SELECT * FROM policy_bundles
WHERE active = TRUE
ORDER BY name ASC;

-- name: UpdatePolicyBundle :one
UPDATE policy_bundles
SET
    name = $2,
    version = $3,
    description = $4,
    rules = $5,
    checksum = $6,
    active = $7,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeletePolicyBundle :exec
DELETE FROM policy_bundles
WHERE id = $1;

-- name: CountPolicyBundles :one
SELECT COUNT(*) FROM policy_bundles;
