-- name: CreateCertificate :one
INSERT INTO certificates (user_id, serial_number, subject_cn, cert_pem, kms_key_id, not_before, not_after)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetCertificateBySerial :one
SELECT * FROM certificates WHERE serial_number = $1;

-- name: GetActiveCertificateByUser :one
SELECT * FROM certificates
WHERE user_id = $1 AND status = 'active'
ORDER BY created_at DESC LIMIT 1;

-- name: RevokeCertificate :exec
UPDATE certificates
SET status = 'revoked', revoked_at = now(), revoke_reason = $2
WHERE id = $1;