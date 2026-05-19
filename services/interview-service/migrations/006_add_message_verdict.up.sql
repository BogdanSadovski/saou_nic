-- Persist per-turn AI verdicts in interview_messages.
--
-- The Go code already populates session.Messages[i].Verdict in memory,
-- but the schema was missing the columns — verdicts were lost on
-- service restart and never surfaced in /messages history queries.
-- See README §"Sessions and verdicts" for the verdict vocabulary.

ALTER TABLE interview_messages
  ADD COLUMN IF NOT EXISTS verdict          VARCHAR(16),
  ADD COLUMN IF NOT EXISTS verdict_reason   TEXT;

-- The verdict vocabulary is bounded; guard against drift.
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'chk_interview_messages_verdict'
  ) THEN
    ALTER TABLE interview_messages
      ADD CONSTRAINT chk_interview_messages_verdict
      CHECK (verdict IS NULL OR verdict IN
        ('correct','partial','wrong','skipped','off_topic','none'));
  END IF;
END $$;

-- Partial index for analytics: «найти все skipped/wrong ответы по
-- сессии». Полный индекс по verdict не нужен — большинство строк это
-- AI-сообщения, у которых verdict=NULL.
CREATE INDEX IF NOT EXISTS idx_interview_messages_verdict_user
  ON interview_messages(session_id, verdict)
  WHERE verdict IS NOT NULL;
