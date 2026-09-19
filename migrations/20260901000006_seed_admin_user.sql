-- +goose Up
-- +goose StatementBegin
-- Seed admin default (password: ChangeMe!2026 — HARUS diganti di production).
-- Hash Argon2id dihasilkan saat setup; nilai di bawah adalah placeholder
-- yang valid secara format. Ganti sebelum deploy.
INSERT INTO users (email, password_hash, full_name, role, status, email_verified_at)
VALUES (
    'admin@wesign.local',
    '$argon2id$v=19$m=65536,t=3,p=4$c2VlZHNhbHRzZWVkc2FsdA$CHANGEME_HASH_BEFORE_DEPLOYMENT000000000000000000000000',
    'System Administrator',
    'admin',
    'active',
    now()
)
ON CONFLICT (email) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM users WHERE email = 'admin@wesign.local';
-- +goose StatementEnd