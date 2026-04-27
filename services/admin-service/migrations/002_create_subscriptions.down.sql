-- Drop audit_logs indexes
DROP INDEX IF EXISTS idx_audit_logs_admin_created;
DROP INDEX IF EXISTS idx_audit_logs_created_at;
DROP INDEX IF EXISTS idx_audit_logs_resource_id;
DROP INDEX IF EXISTS idx_audit_logs_resource_type;
DROP INDEX IF EXISTS idx_audit_logs_action;
DROP INDEX IF EXISTS idx_audit_logs_admin_id;

-- Drop audit_logs table
DROP TABLE IF EXISTS audit_logs;

-- Drop subscriptions indexes
DROP INDEX IF EXISTS idx_subscriptions_active;
DROP INDEX IF EXISTS idx_subscriptions_created_at;
DROP INDEX IF EXISTS idx_subscriptions_end_date;
DROP INDEX IF EXISTS idx_subscriptions_status;
DROP INDEX IF EXISTS idx_subscriptions_tier;
DROP INDEX IF EXISTS idx_subscriptions_user_id;

-- Drop trigger
DROP TRIGGER IF EXISTS update_subscriptions_updated_at ON subscriptions;

-- Drop subscriptions table
DROP TABLE IF EXISTS subscriptions;
