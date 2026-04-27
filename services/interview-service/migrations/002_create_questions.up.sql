-- Create questions table
CREATE TABLE IF NOT EXISTS questions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    interview_id UUID NOT NULL REFERENCES interviews(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    type VARCHAR(50) NOT NULL,
    difficulty VARCHAR(50) NOT NULL,
    tags JSONB NOT NULL DEFAULT '[]',
    starter_code TEXT,
    solution TEXT,
    test_cases JSONB NOT NULL DEFAULT '[]',
    points INTEGER NOT NULL DEFAULT 10,
    question_order INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for common queries
CREATE INDEX idx_questions_interview_id ON questions(interview_id);
CREATE INDEX idx_questions_type ON questions(type);
CREATE INDEX idx_questions_difficulty ON questions(difficulty);
CREATE INDEX idx_questions_order ON questions(question_order);

-- Add constraint for valid question types
ALTER TABLE questions ADD CONSTRAINT check_question_type
    CHECK (type IN ('coding', 'system_design', 'behavioral', 'debugging'));

-- Add constraint for valid difficulty levels
ALTER TABLE questions ADD CONSTRAINT check_question_difficulty
    CHECK (difficulty IN ('easy', 'medium', 'hard', 'expert'));

-- Create sessions table
CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    interview_id UUID NOT NULL REFERENCES interviews(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL DEFAULT 'scheduled',
    current_question_index INTEGER NOT NULL DEFAULT 0,
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE,
    score INTEGER NOT NULL DEFAULT 0,
    feedback TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for sessions
CREATE INDEX idx_sessions_interview_id ON sessions(interview_id);
CREATE INDEX idx_sessions_status ON sessions(status);

-- Create answers table
CREATE TABLE IF NOT EXISTS answers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    question_id UUID NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    code TEXT NOT NULL,
    language VARCHAR(50) NOT NULL,
    is_correct BOOLEAN,
    score INTEGER NOT NULL DEFAULT 0,
    submitted_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for answers
CREATE INDEX idx_answers_session_id ON answers(session_id);
CREATE INDEX idx_answers_question_id ON answers(question_id);
CREATE INDEX idx_answers_submitted_at ON answers(submitted_at);
