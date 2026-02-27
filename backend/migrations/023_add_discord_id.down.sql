DROP INDEX IF EXISTS idx_users_discord_id;
ALTER TABLE users DROP COLUMN IF EXISTS discord_id;
