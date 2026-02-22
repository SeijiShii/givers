CREATE TABLE IF NOT EXISTS user_cost_presets (
    id         VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id    VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    label      VARCHAR(100) NOT NULL,
    unit_type  VARCHAR(20) NOT NULL CHECK (unit_type IN ('monthly', 'daily_x_days')),
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_cost_presets_user_id ON user_cost_presets(user_id);
