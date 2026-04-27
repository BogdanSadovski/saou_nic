-- Migration: 001_create_resumes
-- Description: Creates the resumes table for storing parsed resume data

CREATE TABLE IF NOT EXISTS resumes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    file_name VARCHAR(500) NOT NULL,
    file_url TEXT,
    content_type VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',

    -- Extracted contact information
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    email VARCHAR(255),
    phone VARCHAR(50),

    -- Resume content
    summary TEXT,
    skills JSONB DEFAULT '[]',
    experience JSONB DEFAULT '[]',
    education JSONB DEFAULT '[]',
    languages JSONB DEFAULT '[]',
    certifications JSONB DEFAULT '[]',

    -- Metadata
    metadata JSONB DEFAULT '{}',

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Error information
    error TEXT
);

-- Index for user-based queries
CREATE INDEX idx_resumes_user_id ON resumes(user_id);

-- Index for status-based filtering
CREATE INDEX idx_resumes_status ON resumes(status);

-- Index for creation date sorting
CREATE INDEX idx_resumes_created_at ON resumes(created_at DESC);

-- Composite index for common query pattern
CREATE INDEX idx_resumes_user_status ON resumes(user_id, status);

-- Add comment for documentation
COMMENT ON TABLE resumes IS 'Stores parsed resume documents and their extracted data';
COMMENT ON COLUMN resumes.status IS 'Processing status: pending, processing, completed, failed';
COMMENT ON COLUMN resumes.skills IS 'Array of extracted technical skills';
COMMENT ON COLUMN resumes.experience IS 'Array of work experience entries';
COMMENT ON COLUMN resumes.education IS 'Array of education entries';
COMMENT ON COLUMN resumes.languages IS 'Array of spoken languages';
COMMENT ON COLUMN resumes.certifications IS 'Array of professional certifications';
