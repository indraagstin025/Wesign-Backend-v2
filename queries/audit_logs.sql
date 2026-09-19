-- name: AppendAuditLog :one
INSERT INTO audit_logs (user_id, document_id, action, entity_type, entity_id, ip_address, user_agent, metadata, previous_hash, hash)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetLastAuditLog :one
SELECT * FROM audit_logs ORDER BY id DESC LIMIT 1;

-- name: ListAuditLogsByDocument :many
SELECT * FROM audit_logs
WHERE document_id = $1
ORDER BY id ASC
LIMIT $2 OFFSET $3;

-- name: CreateHashAnchor :one
INSERT INTO hash_anchors (chain_tip_hash, anchor_type, anchor_ref)
VALUES ($1, $2, $3)
RETURNING *;