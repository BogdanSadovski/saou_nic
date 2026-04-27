-- AI interview module data models

CREATE TABLE IF NOT EXISTS interview_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    role VARCHAR(120) NOT NULL,
    level VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'created',
    current_topic VARCHAR(120) NOT NULL DEFAULT 'intro',
    difficulty_score INTEGER NOT NULL DEFAULT 5,
    pressure_level INTEGER NOT NULL DEFAULT 1,
    question_count INTEGER NOT NULL DEFAULT 0,
    question_limit INTEGER NOT NULL DEFAULT 10,
    started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMP WITH TIME ZONE,
    duration_seconds INTEGER NOT NULL DEFAULT 1800,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_interview_sessions_user_id ON interview_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_interview_sessions_status ON interview_sessions(status);
CREATE INDEX IF NOT EXISTS idx_interview_sessions_started_at ON interview_sessions(started_at DESC);

ALTER TABLE interview_sessions DROP CONSTRAINT IF EXISTS chk_interview_sessions_level;
ALTER TABLE interview_sessions
    ADD CONSTRAINT chk_interview_sessions_level
    CHECK (level IN ('junior', 'middle', 'senior'));

ALTER TABLE interview_sessions DROP CONSTRAINT IF EXISTS chk_interview_sessions_status;
ALTER TABLE interview_sessions
    ADD CONSTRAINT chk_interview_sessions_status
    CHECK (status IN ('created', 'active', 'finished', 'failed'));

ALTER TABLE interview_sessions DROP CONSTRAINT IF EXISTS chk_interview_sessions_difficulty;
ALTER TABLE interview_sessions
    ADD CONSTRAINT chk_interview_sessions_difficulty
    CHECK (difficulty_score >= 1 AND difficulty_score <= 10);

ALTER TABLE interview_sessions DROP CONSTRAINT IF EXISTS chk_interview_sessions_pressure;
ALTER TABLE interview_sessions
    ADD CONSTRAINT chk_interview_sessions_pressure
    CHECK (pressure_level >= 1 AND pressure_level <= 5);

ALTER TABLE interview_sessions DROP CONSTRAINT IF EXISTS chk_interview_sessions_question_limit;
ALTER TABLE interview_sessions
    ADD CONSTRAINT chk_interview_sessions_question_limit
    CHECK (question_limit >= 1 AND question_limit <= 200);

ALTER TABLE interview_sessions DROP CONSTRAINT IF EXISTS chk_interview_sessions_question_count;
ALTER TABLE interview_sessions
    ADD CONSTRAINT chk_interview_sessions_question_count
    CHECK (question_count >= 0 AND question_count <= question_limit);

ALTER TABLE interview_sessions DROP CONSTRAINT IF EXISTS chk_interview_sessions_duration;
ALTER TABLE interview_sessions
    ADD CONSTRAINT chk_interview_sessions_duration
    CHECK (duration_seconds >= 60 AND duration_seconds <= 21600);

CREATE TABLE IF NOT EXISTS interview_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES interview_sessions(id) ON DELETE CASCADE,
    sender VARCHAR(16) NOT NULL,
    content TEXT NOT NULL,
    topic VARCHAR(120),
    difficulty INTEGER,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    token_usage JSONB
);

CREATE INDEX IF NOT EXISTS idx_interview_messages_session_created ON interview_messages(session_id, created_at);
CREATE INDEX IF NOT EXISTS idx_interview_messages_sender ON interview_messages(sender);

ALTER TABLE interview_messages DROP CONSTRAINT IF EXISTS chk_interview_messages_sender;
ALTER TABLE interview_messages
    ADD CONSTRAINT chk_interview_messages_sender
    CHECK (sender IN ('ai', 'user', 'system'));

ALTER TABLE interview_messages DROP CONSTRAINT IF EXISTS chk_interview_messages_difficulty;
ALTER TABLE interview_messages
    ADD CONSTRAINT chk_interview_messages_difficulty
    CHECK (difficulty IS NULL OR (difficulty >= 1 AND difficulty <= 10));

CREATE TABLE IF NOT EXISTS interview_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL UNIQUE REFERENCES interview_sessions(id) ON DELETE CASCADE,
    correctness NUMERIC(5,2) NOT NULL,
    clarity NUMERIC(5,2) NOT NULL,
    completeness NUMERIC(5,2) NOT NULL,
    relevance NUMERIC(5,2) NOT NULL,
    overall_score NUMERIC(5,2) NOT NULL,
    strengths JSONB[] NOT NULL DEFAULT ARRAY[]::jsonb[],
    weaknesses JSONB[] NOT NULL DEFAULT ARRAY[]::jsonb[],
    recommendations JSONB[] NOT NULL DEFAULT ARRAY[]::jsonb[],
    generated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

ALTER TABLE interview_reports DROP CONSTRAINT IF EXISTS chk_interview_reports_correctness;
ALTER TABLE interview_reports
    ADD CONSTRAINT chk_interview_reports_correctness
    CHECK (correctness >= 0 AND correctness <= 100);

ALTER TABLE interview_reports DROP CONSTRAINT IF EXISTS chk_interview_reports_clarity;
ALTER TABLE interview_reports
    ADD CONSTRAINT chk_interview_reports_clarity
    CHECK (clarity >= 0 AND clarity <= 100);

ALTER TABLE interview_reports DROP CONSTRAINT IF EXISTS chk_interview_reports_completeness;
ALTER TABLE interview_reports
    ADD CONSTRAINT chk_interview_reports_completeness
    CHECK (completeness >= 0 AND completeness <= 100);

ALTER TABLE interview_reports DROP CONSTRAINT IF EXISTS chk_interview_reports_relevance;
ALTER TABLE interview_reports
    ADD CONSTRAINT chk_interview_reports_relevance
    CHECK (relevance >= 0 AND relevance <= 100);

ALTER TABLE interview_reports DROP CONSTRAINT IF EXISTS chk_interview_reports_overall_score;
ALTER TABLE interview_reports
    ADD CONSTRAINT chk_interview_reports_overall_score
    CHECK (overall_score >= 0 AND overall_score <= 100);

CREATE TABLE IF NOT EXISTS request_log (
    id BIGSERIAL PRIMARY KEY,
    idempotency_key VARCHAR(128) NOT NULL UNIQUE,
    session_id UUID REFERENCES interview_sessions(id) ON DELETE CASCADE,
    response_hash VARCHAR(128) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_request_log_session_created ON request_log(session_id, created_at DESC);
