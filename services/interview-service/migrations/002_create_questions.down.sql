-- Drop answers table
DROP INDEX IF EXISTS idx_answers_submitted_at;
DROP INDEX IF EXISTS idx_answers_question_id;
DROP INDEX IF EXISTS idx_answers_session_id;
DROP TABLE IF EXISTS answers;

-- Drop sessions table
DROP INDEX IF EXISTS idx_sessions_status;
DROP INDEX IF EXISTS idx_sessions_interview_id;
DROP TABLE IF EXISTS sessions;

-- Drop questions table
DROP INDEX IF EXISTS idx_questions_order;
DROP INDEX IF EXISTS idx_questions_difficulty;
DROP INDEX IF EXISTS idx_questions_type;
DROP INDEX IF EXISTS idx_questions_interview_id;
DROP TABLE IF EXISTS questions;
