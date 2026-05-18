-- Rollback: Remove columns from users table
ALTER TABLE users DROP CONSTRAINT IF EXISTS check_provider;
ALTER TABLE users ADD CONSTRAINT check_provider CHECK (provider IN ('local', 'google', 'github'));

DROP INDEX IF EXISTS idx_users_github_url;
DROP INDEX IF EXISTS idx_users_login;
DROP INDEX IF EXISTS idx_users_login_local_unique;

ALTER TABLE users DROP COLUMN IF EXISTS login;
ALTER TABLE users DROP COLUMN IF EXISTS phone;
ALTER TABLE users DROP COLUMN IF EXISTS github_url;
ALTER TABLE users DROP COLUMN IF EXISTS github_profile;
ALTER TABLE users DROP COLUMN IF EXISTS resume_url;
ALTER TABLE users DROP COLUMN IF EXISTS bio;
ALTER TABLE users DROP COLUMN IF EXISTS location;
