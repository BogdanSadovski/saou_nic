-- Make user_service.users compatible with admin-service expectations.
-- Admin-service reads from the same users table now (consolidated), so
-- it needs deleted_at (soft-delete filter) and two_factor_enabled
-- columns. Both default to safe values so existing rows light up
-- without backfill.

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS deleted_at         timestamp with time zone NULL,
    ADD COLUMN IF NOT EXISTS two_factor_enabled boolean                  NOT NULL DEFAULT false;

CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users (deleted_at) WHERE deleted_at IS NULL;
