-- Add new fields to users table for enhanced profile management
ALTER TABLE users ADD COLUMN IF NOT EXISTS login VARCHAR(100);
ALTER TABLE users ADD COLUMN IF NOT EXISTS phone VARCHAR(20);
ALTER TABLE users ADD COLUMN IF NOT EXISTS github_url TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS github_profile TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS resume_url TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS bio TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS location VARCHAR(255);

-- Add unique login index for local accounts only
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_login_local_unique ON users(login)
WHERE provider = 'local' AND login IS NOT NULL;

-- Create indexes for new fields
CREATE INDEX IF NOT EXISTS idx_users_github_url ON users(github_url);
CREATE INDEX IF NOT EXISTS idx_users_login ON users(login);

-- Update provider constraint to include google
ALTER TABLE users DROP CONSTRAINT IF EXISTS check_provider;
ALTER TABLE users ADD CONSTRAINT check_provider CHECK (provider IN ('local', 'google', 'github'));
