-- name: CreateUser :one
INSERT INTO users (email, password_hash, full_name, role, status)
VALUES ($1, $2, $3, $4, 'unverified')
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: VerifyUserEmail :exec
UPDATE users
SET status = 'active', email_verified_at = now()
WHERE id = $1;

-- name: UpdateUserPassword :exec
UPDATE users SET password_hash = $2, updated_at = now()
WHERE id = $1;

-- name: CreateSession :one
INSERT INTO user_sessions (user_id, refresh_token_hash, token_family, ip_address, user_agent, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: RevokeSession :exec
UPDATE user_sessions SET revoked_at = now() WHERE id = $1;

-- name: RevokeTokenFamily :exec
UPDATE user_sessions SET revoked_at = now()
WHERE token_family = $1 AND revoked_at IS NULL;