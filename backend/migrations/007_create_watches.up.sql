CREATE TABLE IF NOT EXISTS watches (
    user_id    VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    project_id VARCHAR(36) NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, project_id)
);

CREATE INDEX IF NOT EXISTS idx_watches_user_id    ON watches(user_id);
CREATE INDEX IF NOT EXISTS idx_watches_project_id ON watches(project_id);
