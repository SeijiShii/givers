-- subscriptions: サブスクリプション管理テーブル（pause/resume/cancel/金額変更）
CREATE TABLE IF NOT EXISTS subscriptions (
    id                      VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    project_id              VARCHAR(36) NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    donor_type              VARCHAR(10) NOT NULL CHECK (donor_type IN ('token', 'user')),
    donor_id                TEXT NOT NULL,
    amount                  INTEGER NOT NULL CHECK (amount > 0),
    currency                VARCHAR(3) NOT NULL DEFAULT 'jpy',
    message                 TEXT,
    stripe_subscription_id  TEXT UNIQUE,
    paused                  BOOLEAN NOT NULL DEFAULT false,
    next_billing_message    TEXT,
    created_at              TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_project_id ON subscriptions(project_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_donor ON subscriptions(donor_type, donor_id);

-- 既存 donations テーブルから recurring レコードを移行
INSERT INTO subscriptions (id, project_id, donor_type, donor_id, amount, currency, message, stripe_subscription_id, paused, next_billing_message, created_at, updated_at)
SELECT id, project_id, donor_type, donor_id, amount, currency, message, stripe_subscription_id, paused, next_billing_message, created_at, updated_at
FROM donations
WHERE is_recurring = true AND stripe_subscription_id IS NOT NULL
ON CONFLICT DO NOTHING;
