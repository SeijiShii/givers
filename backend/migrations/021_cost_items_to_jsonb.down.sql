-- project_cost_items テーブル再作成（データ復元は困難）
CREATE TABLE IF NOT EXISTS project_cost_items (
    id             VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    project_id     VARCHAR(36) NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    label          VARCHAR(100) NOT NULL,
    unit_type      VARCHAR(20) NOT NULL CHECK (unit_type IN ('monthly', 'daily_x_days')),
    amount_monthly INT NOT NULL DEFAULT 0,
    rate_per_day   INT NOT NULL DEFAULT 0,
    days_per_month INT NOT NULL DEFAULT 0,
    sort_order     INT NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_project_cost_items_project_id ON project_cost_items(project_id);

ALTER TABLE projects DROP COLUMN IF EXISTS cost_items;
