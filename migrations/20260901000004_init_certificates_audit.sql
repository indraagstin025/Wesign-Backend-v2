-- +goose Up
-- +goose StatementBegin
CREATE TABLE certificates (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    serial_number   VARCHAR(64) NOT NULL UNIQUE,
    subject_cn      VARCHAR(255) NOT NULL,
    cert_pem        TEXT NOT NULL,
    kms_key_id      TEXT NOT NULL,
    not_before      TIMESTAMPTZ NOT NULL,
    not_after       TIMESTAMPTZ NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'active',
    revoked_at      TIMESTAMPTZ,
    revoke_reason   VARCHAR(50),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE audit_logs (
    id              BIGSERIAL PRIMARY KEY,
    user_id         UUID REFERENCES users(id),
    document_id     UUID REFERENCES documents(id),
    action          VARCHAR(50) NOT NULL,
    entity_type     VARCHAR(50),
    entity_id       TEXT,
    ip_address      INET,
    user_agent      TEXT,
    metadata        JSONB,
    previous_hash   CHAR(64) NOT NULL,
    hash            CHAR(64) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_logs_created_desc ON audit_logs(created_at DESC);
CREATE INDEX idx_audit_logs_document_id ON audit_logs(document_id);

CREATE TABLE hash_anchors (
    id              BIGSERIAL PRIMARY KEY,
    chain_tip_hash  CHAR(64) NOT NULL,
    anchor_type     VARCHAR(30) NOT NULL,
    anchor_ref      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS hash_anchors;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS certificates;
-- +goose StatementEnd