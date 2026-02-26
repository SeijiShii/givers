-- =============================================================================
-- GIVErS 全テーブル DROP（リセット用）
-- 依存関係の逆順で削除する。
-- =============================================================================

DROP TABLE IF EXISTS sessions           CASCADE;
DROP TABLE IF EXISTS activities          CASCADE;
DROP TABLE IF EXISTS user_cost_presets   CASCADE;
DROP TABLE IF EXISTS donations           CASCADE;
DROP TABLE IF EXISTS project_updates     CASCADE;
DROP TABLE IF EXISTS watches             CASCADE;
DROP TABLE IF EXISTS project_alerts      CASCADE;
DROP TABLE IF EXISTS contact_messages    CASCADE;
DROP TABLE IF EXISTS platform_health     CASCADE;
DROP TABLE IF EXISTS project_cost_items  CASCADE;
DROP TABLE IF EXISTS project_costs       CASCADE;
DROP TABLE IF EXISTS projects            CASCADE;
DROP TABLE IF EXISTS users               CASCADE;
DROP TABLE IF EXISTS schema_migrations   CASCADE;
