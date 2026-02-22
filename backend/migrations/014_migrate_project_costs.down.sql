-- project_cost_items から project_costs を再作成（ロールバック用）
CREATE TABLE IF NOT EXISTS project_costs (
    id                VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    project_id        VARCHAR(36) NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    server_cost_monthly INT NOT NULL DEFAULT 0,
    dev_cost_per_day    INT NOT NULL DEFAULT 0,
    dev_days_per_month  INT NOT NULL DEFAULT 0,
    other_cost_monthly  INT NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(project_id)
);
-- ※ データの完全復元は困難なため、空テーブルのみ再作成する
