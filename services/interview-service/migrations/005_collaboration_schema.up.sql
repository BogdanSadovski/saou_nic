-- Feature #3: Multi-Interviewer Collaboration Schema

-- Table to track collaborators on interview sessions
CREATE TABLE IF NOT EXISTS interview_collaborators (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES interview_sessions(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    role VARCHAR(50) NOT NULL CHECK (role IN ('lead', 'observer', 'co-interviewer')),
    joined_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    left_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_collaborators_session ON interview_collaborators(session_id);
CREATE INDEX IF NOT EXISTS idx_collaborators_user ON interview_collaborators(user_id);
CREATE INDEX IF NOT EXISTS idx_collaborators_active ON interview_collaborators(is_active);

-- Real-time notes shared between interviewers
CREATE TABLE IF NOT EXISTS collaboration_notes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES interview_sessions(id) ON DELETE CASCADE,
    author_id UUID NOT NULL,
    content TEXT NOT NULL,
    version INT NOT NULL DEFAULT 1,
    is_pinned BOOLEAN DEFAULT FALSE,
    mentions JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notes_session ON collaboration_notes(session_id);
CREATE INDEX IF NOT EXISTS idx_notes_author ON collaboration_notes(author_id);
CREATE INDEX IF NOT EXISTS idx_notes_created ON collaboration_notes(created_at DESC);

-- Independent scoring by each interviewer
CREATE TABLE IF NOT EXISTS interviewer_scores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES interview_sessions(id) ON DELETE CASCADE,
    interviewer_id UUID NOT NULL,
    
    -- Category scores (1-10)
    technical_score INT CHECK (technical_score >= 0 AND technical_score <= 10),
    communication_score INT CHECK (communication_score >= 0 AND communication_score <= 10),
    problem_solving_score INT CHECK (problem_solving_score >= 0 AND problem_solving_score <= 10),
    culture_fit_score INT CHECK (culture_fit_score >= 0 AND culture_fit_score <= 10),
    coding_quality_score INT CHECK (coding_quality_score >= 0 AND coding_quality_score <= 10),
    
    -- Overall recommendation
    recommendation VARCHAR(20) CHECK (recommendation IN ('STRONG_YES', 'YES', 'MAYBE', 'NO', 'STRONG_NO')),
    
    -- Detailed feedback
    strengths TEXT,
    areas_for_improvement TEXT,
    additional_comments TEXT,
    
    submitted_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_scores_session ON interviewer_scores(session_id);
CREATE INDEX IF NOT EXISTS idx_scores_interviewer ON interviewer_scores(interviewer_id);
CREATE INDEX IF NOT EXISTS idx_scores_submitted ON interviewer_scores(submitted_at);

-- Consensus/aggregated scoring
CREATE TABLE IF NOT EXISTS interview_consensus (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES interview_sessions(id) ON DELETE CASCADE UNIQUE,
    
    -- Average scores
    avg_technical_score NUMERIC(3,1),
    avg_communication_score NUMERIC(3,1),
    avg_problem_solving_score NUMERIC(3,1),
    avg_culture_fit_score NUMERIC(3,1),
    avg_coding_quality_score NUMERIC(3,1),
    
    -- Score variance (measure of disagreement)
    score_variance NUMERIC(5,2),
    disagreement_level VARCHAR(20) CHECK (disagreement_level IN ('LOW', 'MEDIUM', 'HIGH')),
    
    -- Consensus recommendation
    consensus_recommendation VARCHAR(20),
    confidence_score NUMERIC(3,2),
    
    -- Which interviewers agreed/disagreed
    alignments JSONB,
    
    calculated_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_consensus_session ON interview_consensus(session_id);

-- Track score changes for audit trail
CREATE TABLE IF NOT EXISTS score_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL,
    interviewer_id UUID NOT NULL,
    action VARCHAR(50) NOT NULL CHECK (action IN ('CREATED', 'UPDATED', 'SUBMITTED', 'DELETED')),
    old_scores JSONB,
    new_scores JSONB,
    change_reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_session ON score_audit_log(session_id);
CREATE INDEX IF NOT EXISTS idx_audit_interviewer ON score_audit_log(interviewer_id);
