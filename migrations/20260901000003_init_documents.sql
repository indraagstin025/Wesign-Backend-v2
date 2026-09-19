-- +goose Up
-- +goose StatementBegin
CREATE TABLE packages (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        VARCHAR(255) NOT NULL,
    status      VARCHAR(20) NOT NULL DEFAULT 'draft',
    expires_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_packages_user_status ON packages(user_id, status);

CREATE TABLE documents (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    package_id      UUID REFERENCES packages(id) ON DELETE SET NULL,
    filename        VARCHAR(255) NOT NULL,
    original_hash   CHAR(64) NOT NULL,
    size_bytes      BIGINT NOT NULL,
    page_count      INT,
    s3_key          TEXT NOT NULL,
    workflow        VARCHAR(20) NOT NULL DEFAULT 'personal',
    status          VARCHAR(20) NOT NULL DEFAULT 'draft',
    expires_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_documents_user_status ON documents(user_id, status);
CREATE INDEX idx_documents_original_hash ON documents(original_hash);
CREATE INDEX idx_documents_package_id ON documents(package_id);

CREATE TABLE document_versions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id     UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    version         INT NOT NULL,
    original_hash   CHAR(64) NOT NULL,
    s3_key          TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(document_id, version)
);

CREATE TABLE signature_assets (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    asset_type  VARCHAR(20) NOT NULL,
    s3_key      TEXT NOT NULL,
    is_default  BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE signature_placements (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id     UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    asset_id        UUID REFERENCES signature_assets(id),
    signer_email    VARCHAR(254),
    page_number     INT NOT NULL,
    x_coordinate    DOUBLE PRECISION NOT NULL,
    y_coordinate    DOUBLE PRECISION NOT NULL,
    width           DOUBLE PRECISION NOT NULL,
    height          DOUBLE PRECISION NOT NULL,
    sequence_order  INT NOT NULL DEFAULT 1,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE signing_requests (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id     UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    signer_email    VARCHAR(254) NOT NULL,
    signer_name     VARCHAR(255),
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    sequence_order  INT NOT NULL DEFAULT 1,
    invite_token    TEXT NOT NULL UNIQUE,
    signed_at       TIMESTAMPTZ,
    declined_at     TIMESTAMPTZ,
    decline_reason  TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_signing_requests_doc_seq ON signing_requests(document_id, sequence_order);
CREATE INDEX idx_signing_requests_email_status ON signing_requests(signer_email, status);

CREATE TABLE otp_codes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    signing_request_id UUID REFERENCES signing_requests(id) ON DELETE CASCADE,
    user_id         UUID REFERENCES users(id) ON DELETE CASCADE,
    code_hash       TEXT NOT NULL,
    purpose         VARCHAR(30) NOT NULL,
    attempts        INT NOT NULL DEFAULT 0,
    max_attempts    INT NOT NULL DEFAULT 5,
    expires_at      TIMESTAMPTZ NOT NULL,
    used_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS otp_codes;
DROP TABLE IF EXISTS signing_requests;
DROP TABLE IF EXISTS signature_placements;
DROP TABLE IF EXISTS signature_assets;
DROP TABLE IF EXISTS document_versions;
DROP TABLE IF EXISTS documents;
DROP TABLE IF EXISTS packages;
-- +goose StatementEnd