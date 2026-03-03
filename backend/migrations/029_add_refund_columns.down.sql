DROP INDEX IF EXISTS idx_donations_refund_status;
ALTER TABLE donations DROP COLUMN IF EXISTS stripe_refund_id;
ALTER TABLE donations DROP COLUMN IF EXISTS refund_status;
