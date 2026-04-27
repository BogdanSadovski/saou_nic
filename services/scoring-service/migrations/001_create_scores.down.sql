-- Drop foreign key constraint
ALTER TABLE scores DROP CONSTRAINT IF EXISTS fk_scores_rubric;

-- Drop indexes
DROP INDEX IF EXISTS idx_scores_submission_id;
DROP INDEX IF EXISTS idx_scores_score_type;
DROP INDEX IF EXISTS idx_scores_status;
DROP INDEX IF EXISTS idx_scores_created_at;
DROP INDEX IF EXISTS idx_rubrics_score_type;

-- Drop tables
DROP TABLE IF EXISTS scores;
DROP TABLE IF EXISTS rubrics;
