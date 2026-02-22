CREATE UNIQUE INDEX IF NOT EXISTS idx_donations_stripe_payment_id_unique
    ON donations(stripe_payment_id) WHERE stripe_payment_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_donations_stripe_subscription_id_unique
    ON donations(stripe_subscription_id) WHERE stripe_subscription_id IS NOT NULL;
