-- +migrate Up
CREATE TYPE reset_subject_type AS ENUM ('USER', 'STAFF');

CREATE TABLE password_reset_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    subject_type reset_subject_type NOT NULL,
    subject_id UUID NOT NULL,
    token VARCHAR(128) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_password_reset_subject ON password_reset_tokens(subject_type, subject_id);
