CREATE TABLE IF NOT EXISTS activities (
    id VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    type VARCHAR(20) NOT NULL CHECK (type IN ('donation', 'project_created', 'project_updated', 'milestone')),
    project_id VARCHAR(36) NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    actor_id VARCHAR(36),
    amount INTEGER,
    rate INTEGER,
    message TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_activities_project_id ON activities(project_id);
CREATE INDEX IF NOT EXISTS idx_activities_created_at ON activities(created_at DESC);
