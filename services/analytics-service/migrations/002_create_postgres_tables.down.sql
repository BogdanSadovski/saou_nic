-- +migrate Down
-- Rollback 002: Create PostgreSQL metadata tables

DROP TABLE IF EXISTS user_sessions CASCADE;
DROP TABLE IF EXISTS export_requests CASCADE;
DROP TABLE IF EXISTS funnels CASCADE;
DROP TABLE IF EXISTS dashboards CASCADE;
