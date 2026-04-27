-- Migration: 001_create_resumes (down)
-- Description: Drops the resumes table

DROP INDEX IF EXISTS idx_resumes_user_status;
DROP INDEX IF EXISTS idx_resumes_created_at;
DROP INDEX IF EXISTS idx_resumes_status;
DROP INDEX IF EXISTS idx_resumes_user_id;

DROP TABLE IF EXISTS resumes;
