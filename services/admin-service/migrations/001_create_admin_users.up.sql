-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL UNIQUE,
    username VARCHAR(100) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'user' CHECK (role IN ('super_admin', 'admin', 'moderator', 'user')),
    status VARCHAR(50) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'suspended', 'banned')),
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    avatar_url VARCHAR(500),
    last_login_at TIMESTAMP WITH TIME ZONE,
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    two_factor_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Create indexes for users table
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_created_at ON users(created_at DESC);
CREATE INDEX idx_users_deleted_at ON users(deleted_at) WHERE deleted_at IS NULL;

-- Create index for full-text search
CREATE INDEX idx_users_search ON users USING gin (
    to_tsvector('english', COALESCE(email, '') || ' ' || COALESCE(username, '') || ' ' || COALESCE(first_name, '') || ' ' || COALESCE(last_name, ''))
);

-- Add trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Insert default super admin user (password: change-me-in-production)
INSERT INTO users (id, email, username, password_hash, role, status, first_name, last_name, email_verified, created_at, updated_at)
VALUES (
    gen_random_uuid(),
    'admin@real-ass.local',
    'superadmin',
    '$2a$10$dXJ3SXn6nZ8zKxQv8q9M.OQK5p4r5t6u7v8w9x0y1z2a3b4c5d6e7f8',
    'super_admin',
    'active',
    'Super',
    'Admin',
    TRUE,
    NOW(),
    NOW()
) ON CONFLICT (email) DO NOTHING;
