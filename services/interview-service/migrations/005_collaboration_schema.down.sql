-- Rollback: Feature #3 Multi-Interviewer Collaboration

DROP TABLE IF EXISTS score_audit_log CASCADE;
DROP TABLE IF EXISTS interview_consensus CASCADE;
DROP TABLE IF EXISTS interviewer_scores CASCADE;
DROP TABLE IF EXISTS collaboration_notes CASCADE;
DROP TABLE IF EXISTS interview_collaborators CASCADE;
