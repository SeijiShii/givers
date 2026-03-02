DROP INDEX IF EXISTS idx_donations_stripe_invoice_id_unique;
DROP INDEX IF EXISTS idx_donations_subscription_id;
DROP INDEX IF EXISTS idx_donations_source;
ALTER TABLE donations DROP COLUMN IF EXISTS stripe_invoice_id;
ALTER TABLE donations DROP COLUMN IF EXISTS subscription_id;
ALTER TABLE donations DROP COLUMN IF EXISTS source;
