-- +goose Up
-- +goose StatementBegin
CREATE TABLE user_reports (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reporter_id     UUID REFERENCES users(id),
    reported_user_id UUID REFERENCES users(id),
    document_id     UUID REFERENCES documents(id),
    reason          TEXT NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'open',
    resolution      TEXT,
    handled_by      UUID REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at     TIMESTAMPTZ
);

CREATE TABLE data_requests (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    request_type    VARCHAR(20) NOT NULL, -- access | correction | deletion | portability
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    details         TEXT,
    result          TEXT,
    handled_by      UUID REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at    TIMESTAMPTZ
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS data_requests;
DROP TABLE IF EXISTS user_reports;
-- +goose StatementEnd