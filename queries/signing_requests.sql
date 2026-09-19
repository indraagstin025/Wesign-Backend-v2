-- name: ListSigningRequestsByDocument :many
SELECT * FROM signing_requests
WHERE document_id = $1
ORDER BY sequence_order ASC;

-- name: ListPendingRequestsByDocument :many
SELECT * FROM signing_requests
WHERE document_id = $1 AND status = 'pending'
ORDER BY sequence_order ASC;

-- name: CountSignedRequests :one
SELECT COUNT(*) FROM signing_requests
WHERE document_id = $1 AND status = 'signed';

-- name: CreateOTP :one
INSERT INTO otp_codes (signing_request_id, code_hash, purpose, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetOTPByID :one
SELECT * FROM otp_codes WHERE id = $1;

-- name: IncrementOTPAttempts :exec
UPDATE otp_codes SET attempts = attempts + 1 WHERE id = $1;

-- name: MarkOTPUsed :exec
UPDATE otp_codes SET used_at = now() WHERE id = $1;