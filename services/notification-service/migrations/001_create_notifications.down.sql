-- Drop trigger first
DROP TRIGGER IF EXISTS update_notifications_updated_at ON notifications;

-- Drop the trigger function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_notifications_pending;
DROP INDEX IF EXISTS idx_notifications_channel;
DROP INDEX IF EXISTS idx_notifications_type;
DROP INDEX IF EXISTS idx_notifications_user_created;
DROP INDEX IF EXISTS idx_notifications_created_at;
DROP INDEX IF EXISTS idx_notifications_status;
DROP INDEX IF EXISTS idx_notifications_user_id;

-- Drop the table
DROP TABLE IF EXISTS notifications;
