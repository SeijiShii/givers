ALTER TABLE users ADD COLUMN stripe_customer_id VARCHAR(255);
CREATE UNIQUE INDEX idx_users_stripe_customer_id ON users (stripe_customer_id) WHERE stripe_customer_id IS NOT NULL;
