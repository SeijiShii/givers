CREATE TABLE IF NOT EXISTS message_threads (
    id VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    host_user_id VARCHAR(36) NOT NULL REFERENCES users(id),
    participant_user_id VARCHAR(36) NOT NULL REFERENCES users(id),
    subject VARCHAR(200) NOT NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_message_threads_host
    ON message_threads(host_user_id);
CREATE INDEX IF NOT EXISTS idx_message_threads_participant
    ON message_threads(participant_user_id);

CREATE TABLE IF NOT EXISTS messages (
    id VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    thread_id VARCHAR(36) NOT NULL REFERENCES message_threads(id) ON DELETE CASCADE,
    sender_id VARCHAR(36) NOT NULL REFERENCES users(id),
    body TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_messages_thread_created
    ON messages(thread_id, created_at DESC);

CREATE TABLE IF NOT EXISTS message_reads (
    user_id VARCHAR(36) NOT NULL REFERENCES users(id),
    thread_id VARCHAR(36) NOT NULL REFERENCES message_threads(id) ON DELETE CASCADE,
    last_read_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, thread_id)
);
