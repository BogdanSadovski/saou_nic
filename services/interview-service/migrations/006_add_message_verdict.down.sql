DROP INDEX IF EXISTS idx_interview_messages_verdict_user;
ALTER TABLE interview_messages
  DROP CONSTRAINT IF EXISTS chk_interview_messages_verdict;
ALTER TABLE interview_messages
  DROP COLUMN IF EXISTS verdict,
  DROP COLUMN IF EXISTS verdict_reason;
