ALTER TABLE users ADD COLUMN IF NOT EXISTS discord_id VARCHAR(255) UNIQUE;
CREATE INDEX IF NOT EXISTS idx_users_discord_id ON users(discord_id);
