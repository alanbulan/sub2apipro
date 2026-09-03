CREATE TABLE IF NOT EXISTS conversation_logs (
    id BIGSERIAL PRIMARY KEY,
    request_id VARCHAR(128) NOT NULL,
    session_id VARCHAR(255) NOT NULL,
    user_id BIGINT NOT NULL,
    user_email VARCHAR(255) NOT NULL DEFAULT '',
    api_key_id BIGINT NOT NULL,
    api_key_name VARCHAR(255) NOT NULL DEFAULT '',
    group_id BIGINT NULL,
    protocol VARCHAR(64) NOT NULL,
    endpoint VARCHAR(255) NOT NULL,
    model VARCHAR(255) NOT NULL DEFAULT '',
    stream BOOLEAN NOT NULL DEFAULT FALSE,
    status VARCHAR(24) NOT NULL DEFAULT 'pending',
    status_code INTEGER NOT NULL DEFAULT 0,
    content_type VARCHAR(128) NOT NULL DEFAULT '',
    request_body TEXT NOT NULL DEFAULT '',
    response_body TEXT NOT NULL DEFAULT '',
    request_bytes BIGINT NOT NULL DEFAULT 0,
    response_bytes BIGINT NOT NULL DEFAULT 0,
    request_truncated BOOLEAN NOT NULL DEFAULT FALSE,
    response_truncated BOOLEAN NOT NULL DEFAULT FALSE,
    duration_ms BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ NULL,
    CONSTRAINT conversation_logs_status_check
        CHECK (status IN ('pending', 'completed', 'failed', 'incomplete')),
    CONSTRAINT conversation_logs_size_check
        CHECK (request_bytes >= 0 AND response_bytes >= 0 AND duration_ms >= 0)
);

CREATE INDEX IF NOT EXISTS idx_conversation_logs_request_id
    ON conversation_logs (request_id);
CREATE INDEX IF NOT EXISTS idx_conversation_logs_created
    ON conversation_logs (created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_conversation_logs_session_created
    ON conversation_logs (session_id, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_conversation_logs_user_created
    ON conversation_logs (user_id, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_conversation_logs_api_key_created
    ON conversation_logs (api_key_id, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_conversation_logs_status_created
    ON conversation_logs (status, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_conversation_logs_protocol_created
    ON conversation_logs (protocol, created_at DESC, id DESC);
