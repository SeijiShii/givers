CREATE TABLE IF NOT EXISTS announcements (
    id VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    author_id VARCHAR(36) NOT NULL REFERENCES users(id),
    title VARCHAR(200) NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    severity VARCHAR(10) NOT NULL DEFAULT 'info'
        CHECK (severity IN ('info', 'warn', 'error')),
    visible BOOLEAN NOT NULL DEFAULT true,
    published_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_announcements_visible_published
    ON announcements(visible, published_at DESC);

CREATE TABLE IF NOT EXISTS announcement_reads (
    user_id VARCHAR(36) NOT NULL REFERENCES users(id),
    announcement_id VARCHAR(36) NOT NULL REFERENCES announcements(id) ON DELETE CASCADE,
    read_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, announcement_id)
);
