-- Create code_submissions table for storing code submissions during interviews
CREATE TABLE IF NOT EXISTS code_submissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES interview_sessions(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    language VARCHAR(50) NOT NULL,
    code TEXT NOT NULL,
    input_data TEXT,
    submission_sequence INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_code_submissions_session ON code_submissions(session_id);
CREATE INDEX IF NOT EXISTS idx_code_submissions_user ON code_submissions(user_id);

-- Create code_execution_results for storing execution results
CREATE TABLE IF NOT EXISTS code_execution_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    submission_id UUID NOT NULL REFERENCES code_submissions(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL,
    output TEXT,
    error_message TEXT,
    execution_time_ms BIGINT,
    memory_used_bytes BIGINT,
    exit_code INT,
    test_results JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_results_submission ON code_execution_results(submission_id);
CREATE INDEX IF NOT EXISTS idx_results_status ON code_execution_results(status);

-- Create code_test_cases for storing test cases associated with assignments
CREATE TABLE IF NOT EXISTS code_test_cases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    question_id UUID NOT NULL,
    test_name VARCHAR(255) NOT NULL,
    input_data TEXT NOT NULL,
    expected_output TEXT NOT NULL,
    is_hidden BOOLEAN DEFAULT FALSE,
    sequence INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_test_cases_question ON code_test_cases(question_id);

-- Add coding task metadata column to interview_messages if not exists
ALTER TABLE interview_messages
ADD COLUMN IF NOT EXISTS coding_task JSONB,
ADD COLUMN IF NOT EXISTS test_cases_count INT DEFAULT 0;

-- Add interview_mode to interview_sessions if not exists
ALTER TABLE interview_sessions
ADD COLUMN IF NOT EXISTS interview_mode VARCHAR(50) DEFAULT 'theory';
