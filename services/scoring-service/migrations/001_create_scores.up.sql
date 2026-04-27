-- Create scores table
CREATE TABLE IF NOT EXISTS scores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    submission_id VARCHAR(255) NOT NULL,
    score_type VARCHAR(50) NOT NULL,
    total_score DOUBLE PRECISION NOT NULL DEFAULT 0,
    max_score DOUBLE PRECISION NOT NULL DEFAULT 100,
    percentage DOUBLE PRECISION NOT NULL DEFAULT 0,
    grade VARCHAR(5) NOT NULL DEFAULT 'F',
    breakdown JSONB NOT NULL DEFAULT '[]',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    rubric_id UUID,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create rubrics table
CREATE TABLE IF NOT EXISTS rubrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    score_type VARCHAR(50) NOT NULL,
    criteria JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for common queries
CREATE INDEX idx_scores_submission_id ON scores(submission_id);
CREATE INDEX idx_scores_score_type ON scores(score_type);
CREATE INDEX idx_scores_status ON scores(status);
CREATE INDEX idx_scores_created_at ON scores(created_at DESC);
CREATE INDEX idx_rubrics_score_type ON rubrics(score_type);

-- Add foreign key constraint for rubric_id (optional, can be null)
ALTER TABLE scores ADD CONSTRAINT fk_scores_rubric
    FOREIGN KEY (rubric_id) REFERENCES rubrics(id) ON DELETE SET NULL;
