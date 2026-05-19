-- Helpers for finding stale active sessions (24 such on prod audit).
-- A background sweeper (Go goroutine in interview-service) hourly
-- transitions active sessions past ends_at to 'expired' and emits
-- a zero-score report so users see a clear "session not completed"
-- entry instead of empty placeholders.

CREATE INDEX IF NOT EXISTS idx_interview_sessions_status_ended_at
  ON interview_sessions(status, ended_at)
  WHERE status = 'active';

-- A view aggregating active-but-orphaned sessions; used by the
-- sweeper's SELECT, also handy for debugging. Note the schema uses
-- `started_at + duration_seconds` to compute the real deadline
-- because `ended_at` is only set when a session actually finishes.
CREATE OR REPLACE VIEW v_stale_interview_sessions AS
SELECT
  s.id,
  s.user_id,
  s.role,
  s.metadata->>'interview_mode' AS interview_mode,
  s.status,
  s.started_at,
  s.started_at + (s.duration_seconds * INTERVAL '1 second') AS deadline_at,
  NOW() - (s.started_at + (s.duration_seconds * INTERVAL '1 second')) AS overdue_by
FROM interview_sessions s
WHERE s.status = 'active'
  AND s.started_at + (s.duration_seconds * INTERVAL '1 second') < NOW();
