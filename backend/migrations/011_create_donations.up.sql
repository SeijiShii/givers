CREATE TABLE IF NOT EXISTS donations (
    id VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    project_id VARCHAR(36) NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    donor_type VARCHAR(10) NOT NULL CHECK (donor_type IN ('token', 'user')),
    donor_id VARCHAR(36) NOT NULL,
    amount INTEGER NOT NULL CHECK (amount > 0),
    currency VARCHAR(3) NOT NULL DEFAULT 'jpy',
    message TEXT,
    is_recurring BOOLEAN NOT NULL DEFAULT false,
    stripe_payment_id TEXT,
    stripe_subscription_id TEXT,
    paused BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_donations_project_id ON donations(project_id);
CREATE INDEX IF NOT EXISTS idx_donations_donor ON donations(donor_type, donor_id);
CREATE INDEX IF NOT EXISTS idx_donations_stripe_subscription_id ON donations(stripe_subscription_id) WHERE stripe_subscription_id IS NOT NULL;
