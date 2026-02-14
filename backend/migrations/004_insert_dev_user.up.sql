-- 開発用ダミーユーザー（AUTH_REQUIRED=false 時に使用）
INSERT INTO users (id, email, name, created_at, updated_at)
VALUES ('dev-user-id', 'dev@givers.local', 'Dev User', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;
