-- 001_init.sql
-- Initial schema for the SMS broadcast system.

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- broadcasts holds the top-level broadcast record.
CREATE TABLE IF NOT EXISTS broadcasts (
    id         UUID        PRIMARY KEY,
    name       TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- messages holds individual SMS messages (the outbox).
CREATE TABLE IF NOT EXISTS messages (
    id           UUID        PRIMARY KEY,
    broadcast_id UUID        NOT NULL REFERENCES broadcasts(id),
    to_number    TEXT        NOT NULL,
    body         TEXT        NOT NULL,
    status       TEXT        NOT NULL DEFAULT 'pending',
    provider_id  TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for the outbox-publisher polling query (status = 'pending', order by created_at ASC).
CREATE INDEX IF NOT EXISTS idx_messages_status_created
    ON messages (status, created_at ASC);

-- Index for DLR webhook lookups by provider_id.
CREATE INDEX IF NOT EXISTS idx_messages_provider_id
    ON messages (provider_id)
    WHERE provider_id IS NOT NULL;
