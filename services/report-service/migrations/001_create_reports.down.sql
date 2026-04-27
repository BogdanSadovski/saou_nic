-- Drop trigger and function
DROP TRIGGER IF EXISTS update_reports_updated_at ON reports;
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_reports_status_created;
DROP INDEX IF EXISTS idx_reports_candidate_status;
DROP INDEX IF EXISTS idx_reports_expires_at;
DROP INDEX IF EXISTS idx_reports_created_at;
DROP INDEX IF EXISTS idx_reports_format;
DROP INDEX IF EXISTS idx_reports_type;
DROP INDEX IF EXISTS idx_reports_status;
DROP INDEX IF EXISTS idx_reports_candidate_id;

-- Drop table
DROP TABLE IF EXISTS reports;
