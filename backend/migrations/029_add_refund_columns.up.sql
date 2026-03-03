ALTER TABLE donations ADD COLUMN IF NOT EXISTS refund_status VARCHAR(20);
ALTER TABLE donations ADD COLUMN IF NOT EXISTS stripe_refund_id TEXT;
CREATE INDEX IF NOT EXISTS idx_donations_refund_status
    ON donations(refund_status) WHERE refund_status IS NOT NULL;
