-- name: CreateDocument :one
INSERT INTO documents (user_id, package_id, filename, original_hash, size_bytes, s3_key, workflow, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, 'draft')
RETURNING *;

-- name: GetDocumentByID :one
SELECT * FROM documents WHERE id = $1;

-- name: ListDocumentsByUser :many
SELECT * FROM documents
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetDocumentByHash :one
SELECT * FROM documents WHERE original_hash = $1 LIMIT 1;

-- name: UpdateDocumentStatus :exec
UPDATE documents SET status = $2, updated_at = now()
WHERE id = $1;

-- name: CreateSigningRequest :one
INSERT INTO signing_requests (document_id, signer_email, signer_name, sequence_order, invite_token)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetSigningRequestByToken :one
SELECT * FROM signing_requests WHERE invite_token = $1;

-- name: UpdateSigningRequestStatus :exec
UPDATE signing_requests
SET status = $2, signed_at = $3, declined_at = $4, decline_reason = $5
WHERE id = $1;