-- 手動寄付 vs 自動決済を区別するカラム
ALTER TABLE donations ADD COLUMN IF NOT EXISTS source VARCHAR(30) NOT NULL DEFAULT 'checkout';

-- どのサブスクリプションから生成された決済かを追跡
ALTER TABLE donations ADD COLUMN IF NOT EXISTS subscription_id VARCHAR(36) REFERENCES subscriptions(id) ON DELETE SET NULL;

-- invoice ベースの冪等性確保用
ALTER TABLE donations ADD COLUMN IF NOT EXISTS stripe_invoice_id TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS idx_donations_stripe_invoice_id_unique
    ON donations(stripe_invoice_id) WHERE stripe_invoice_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_donations_subscription_id ON donations(subscription_id) WHERE subscription_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_donations_source ON donations(source);

-- 既存の recurring donations に source を設定
UPDATE donations SET source = 'checkout' WHERE is_recurring = true;

-- subscription_id を紐付け（stripe_subscription_id 経由）
UPDATE donations d SET subscription_id = s.id
FROM subscriptions s
WHERE d.stripe_subscription_id = s.stripe_subscription_id
  AND d.is_recurring = true;
