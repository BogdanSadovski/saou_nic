-- Drop interviews table
DROP INDEX IF EXISTS idx_interviews_scheduled_at;
DROP INDEX IF EXISTS idx_interviews_status;
DROP INDEX IF EXISTS idx_interviews_candidate_id;
DROP INDEX IF EXISTS idx_interviews_interviewer_id;
DROP TABLE IF EXISTS interviews;
