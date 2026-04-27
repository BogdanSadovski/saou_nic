-- Drop code submission tables and cleanup related columns
DROP TABLE IF EXISTS code_execution_results CASCADE;
DROP TABLE IF EXISTS code_test_cases CASCADE;
DROP TABLE IF EXISTS code_submissions CASCADE;

-- Remove added columns
ALTER TABLE interview_messages
DROP COLUMN IF EXISTS coding_task,
DROP COLUMN IF EXISTS test_cases_count;

ALTER TABLE interview_sessions
DROP COLUMN IF EXISTS interview_mode;
