-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
    id            UUID PRIMARY KEY,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'host',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
    id         UUID PRIMARY KEY,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX sessions_expires_at_idx ON sessions (expires_at);

CREATE TABLE conversations (
    id               UUID PRIMARY KEY,
    subject          TEXT NOT NULL,
    status           TEXT NOT NULL DEFAULT 'open',
    visitor_name     TEXT NOT NULL,
    visitor_email    TEXT NOT NULL,
    visitor_token    TEXT NOT NULL UNIQUE,
    unread           BOOLEAN NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_activity_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX conversations_status_activity_idx ON conversations (status, last_activity_at DESC);
CREATE INDEX conversations_visitor_email_idx ON conversations (visitor_email);

CREATE TABLE messages (
    id               UUID PRIMARY KEY,
    conversation_id  UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    direction        TEXT NOT NULL,
    body             TEXT NOT NULL,
    from_addr        TEXT NOT NULL DEFAULT '',
    to_addr          TEXT NOT NULL DEFAULT '',
    email_message_id TEXT NOT NULL DEFAULT '',
    in_reply_to      TEXT NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX messages_conversation_idx ON messages (conversation_id, created_at);
CREATE UNIQUE INDEX messages_email_message_id_idx
    ON messages (email_message_id) WHERE email_message_id <> '';

CREATE TABLE attachments (
    id              UUID PRIMARY KEY,
    -- conversation_id is NULL until the upload is linked to a submitted message.
    conversation_id UUID REFERENCES conversations(id) ON DELETE CASCADE,
    message_id      UUID REFERENCES messages(id) ON DELETE SET NULL,
    filename        TEXT NOT NULL,
    content_type    TEXT NOT NULL,
    size            BIGINT NOT NULL DEFAULT 0,
    storage_key     TEXT NOT NULL UNIQUE,
    sha256          TEXT NOT NULL DEFAULT '',
    upload_status   TEXT NOT NULL DEFAULT 'pending',
    scan_status     TEXT NOT NULL DEFAULT 'pending',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX attachments_conversation_idx ON attachments (conversation_id);
CREATE INDEX attachments_orphans_idx ON attachments (upload_status, created_at);

CREATE TABLE jobs (
    id           UUID PRIMARY KEY,
    type         TEXT NOT NULL,
    payload      JSONB NOT NULL DEFAULT '{}',
    status       TEXT NOT NULL DEFAULT 'pending',
    attempts     INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 5,
    run_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_error   TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX jobs_claim_idx ON jobs (status, run_at);

-- +goose Down
DROP TABLE IF EXISTS jobs;
DROP TABLE IF EXISTS attachments;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS conversations;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
