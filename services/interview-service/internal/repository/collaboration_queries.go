package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/interview-platform/interview-service/internal/domain"
)

// Collaboration Queries

// AddCollaborator adds an interviewer to session
func (r *PostgresRepository) AddCollaborator(ctx context.Context, sessionID, userID uuid.UUID, role domain.CollaboratorRole) (*domain.InterviewCollaborator, error) {
	query := `
		INSERT INTO interview_collaborators (session_id, user_id, role)
		VALUES ($1, $2, $3)
		RETURNING id, session_id, user_id, role, joined_at, left_at, is_active, created_at
	`

	collab := &domain.InterviewCollaborator{}
	err := r.db.QueryRowContext(ctx, query, sessionID, userID, role).Scan(
		&collab.ID,
		&collab.SessionID,
		&collab.UserID,
		&collab.Role,
		&collab.JoinedAt,
		&collab.LeftAt,
		&collab.IsActive,
		&collab.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return collab, nil
}

// ListCollaborators returns active collaborators for session
func (r *PostgresRepository) ListCollaborators(ctx context.Context, sessionID uuid.UUID) ([]*domain.InterviewCollaborator, error) {
	query := `
		SELECT id, session_id, user_id, role, joined_at, left_at, is_active, created_at
		FROM interview_collaborators
		WHERE session_id = $1 AND is_active = true
		ORDER BY joined_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var collaborators []*domain.InterviewCollaborator
	for rows.Next() {
		c := &domain.InterviewCollaborator{}
		err := rows.Scan(
			&c.ID,
			&c.SessionID,
			&c.UserID,
			&c.Role,
			&c.JoinedAt,
			&c.LeftAt,
			&c.IsActive,
			&c.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		collaborators = append(collaborators, c)
	}

	return collaborators, rows.Err()
}

// RemoveCollaborator marks collaborator as left
func (r *PostgresRepository) RemoveCollaborator(ctx context.Context, sessionID, userID uuid.UUID) error {
	query := `
		UPDATE interview_collaborators
		SET is_active = false, left_at = NOW()
		WHERE session_id = $1 AND user_id = $2
	`

	_, err := r.db.ExecContext(ctx, query, sessionID, userID)
	return err
}

// AddNote adds a collaboration note
func (r *PostgresRepository) AddNote(ctx context.Context, note *domain.CollaborationNote) (*domain.CollaborationNote, error) {
	var mentionsJSON []byte
	if len(note.Mentions) > 0 {
		b, _ := json.Marshal(note.Mentions)
		mentionsJSON = b
	}

	query := `
		INSERT INTO collaboration_notes (session_id, author_id, content, mentions)
		VALUES ($1, $2, $3, $4)
		RETURNING id, session_id, author_id, content, version, is_pinned, mentions, created_at, updated_at
	`

	var mentionsStr sql.NullString
	err := r.db.QueryRowContext(ctx, query, note.SessionID, note.AuthorID, note.Content, mentionsJSON).Scan(
		&note.ID,
		&note.SessionID,
		&note.AuthorID,
		&note.Content,
		&note.Version,
		&note.IsPinned,
		&mentionsStr,
		&note.CreatedAt,
		&note.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if mentionsStr.Valid {
		json.Unmarshal([]byte(mentionsStr.String), &note.Mentions)
	}

	return note, nil
}

// ListNotes returns notes for a session
func (r *PostgresRepository) ListNotes(ctx context.Context, sessionID uuid.UUID, limit, offset int) ([]*domain.CollaborationNote, error) {
	query := `
		SELECT id, session_id, author_id, content, version, is_pinned, mentions, created_at, updated_at
		FROM collaboration_notes
		WHERE session_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, sessionID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []*domain.CollaborationNote
	for rows.Next() {
		n := &domain.CollaborationNote{}
		var mentionsStr sql.NullString

		err := rows.Scan(
			&n.ID,
			&n.SessionID,
			&n.AuthorID,
			&n.Content,
			&n.Version,
			&n.IsPinned,
			&mentionsStr,
			&n.CreatedAt,
			&n.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if mentionsStr.Valid {
			json.Unmarshal([]byte(mentionsStr.String), &n.Mentions)
		}

		notes = append(notes, n)
	}

	return notes, rows.Err()
}

// SubmitScore submits interviewer's score
func (r *PostgresRepository) SubmitScore(ctx context.Context, score *domain.InterviewerScore) (*domain.InterviewerScore, error) {
	now := time.Now()
	score.SubmittedAt = &now

	query := `
		INSERT INTO interviewer_scores (
			session_id, interviewer_id, technical_score, communication_score,
			problem_solving_score, culture_fit_score, coding_quality_score,
			recommendation, strengths, areas_for_improvement, additional_comments,
			submitted_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (session_id, interviewer_id) DO UPDATE SET
			technical_score = $3,
			communication_score = $4,
			problem_solving_score = $5,
			culture_fit_score = $6,
			coding_quality_score = $7,
			recommendation = $8,
			strengths = $9,
			areas_for_improvement = $10,
			additional_comments = $11,
			submitted_at = $12,
			updated_at = NOW()
		RETURNING id, session_id, interviewer_id, technical_score, communication_score,
			problem_solving_score, culture_fit_score, coding_quality_score,
			recommendation, strengths, areas_for_improvement, additional_comments,
			submitted_at, created_at, updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		score.SessionID,
		score.InterviewerID,
		score.TechnicalScore,
		score.CommunicationScore,
		score.ProblemSolvingScore,
		score.CultureFitScore,
		score.CodingQualityScore,
		score.Recommendation,
		score.Strengths,
		score.AreasForImprovement,
		score.AdditionalComments,
		score.SubmittedAt,
	).Scan(
		&score.ID,
		&score.SessionID,
		&score.InterviewerID,
		&score.TechnicalScore,
		&score.CommunicationScore,
		&score.ProblemSolvingScore,
		&score.CultureFitScore,
		&score.CodingQualityScore,
		&score.Recommendation,
		&score.Strengths,
		&score.AreasForImprovement,
		&score.AdditionalComments,
		&score.SubmittedAt,
		&score.CreatedAt,
		&score.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return score, nil
}

// GetScores returns all scores for a session
func (r *PostgresRepository) GetScores(ctx context.Context, sessionID uuid.UUID) ([]*domain.InterviewerScore, error) {
	query := `
		SELECT id, session_id, interviewer_id, technical_score, communication_score,
			problem_solving_score, culture_fit_score, coding_quality_score,
			recommendation, strengths, areas_for_improvement, additional_comments,
			submitted_at, created_at, updated_at
		FROM interviewer_scores
		WHERE session_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scores []*domain.InterviewerScore
	for rows.Next() {
		s := &domain.InterviewerScore{}
		err := rows.Scan(
			&s.ID,
			&s.SessionID,
			&s.InterviewerID,
			&s.TechnicalScore,
			&s.CommunicationScore,
			&s.ProblemSolvingScore,
			&s.CultureFitScore,
			&s.CodingQualityScore,
			&s.Recommendation,
			&s.Strengths,
			&s.AreasForImprovement,
			&s.AdditionalComments,
			&s.SubmittedAt,
			&s.CreatedAt,
			&s.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		scores = append(scores, s)
	}

	return scores, rows.Err()
}

// CalculateConsensus calculates consensus from all scores
func (r *PostgresRepository) CalculateConsensus(ctx context.Context, sessionID uuid.UUID) (*domain.InterviewConsensus, error) {
	// Calculate averages and variance
	query := `
		SELECT
			ROUND(AVG(CAST(technical_score AS NUMERIC)), 1),
			ROUND(AVG(CAST(communication_score AS NUMERIC)), 1),
			ROUND(AVG(CAST(problem_solving_score AS NUMERIC)), 1),
			ROUND(AVG(CAST(culture_fit_score AS NUMERIC)), 1),
			ROUND(AVG(CAST(coding_quality_score AS NUMERIC)), 1),
			ROUND(STDDEV(CAST(technical_score AS NUMERIC))::NUMERIC, 2),
			COUNT(DISTINCT interviewer_id),
			COALESCE(MODE() WITHIN GROUP (ORDER BY recommendation), 'MAYBE')
		FROM interviewer_scores
		WHERE session_id = $1 AND submitted_at IS NOT NULL
	`

	consensus := &domain.InterviewConsensus{
		ID:        uuid.New(),
		SessionID: sessionID,
	}

	var avgTech, avgComm, avgProb, avgCult, avgCode *float64
	var variance *float64
	var count int64
	var recommendation string

	err := r.db.QueryRowContext(ctx, query, sessionID).Scan(
		&avgTech,
		&avgComm,
		&avgProb,
		&avgCult,
		&avgCode,
		&variance,
		&count,
		&recommendation,
	)

	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	consensus.AvgTechnicalScore = avgTech
	consensus.AvgCommunicationScore = avgComm
	consensus.AvgProblemSolvingScore = avgProb
	consensus.AvgCultureFitScore = avgCult
	consensus.AvgCodingQualityScore = avgCode
	consensus.ScoreVariance = variance
	consensus.ConsensusRecommendation = &recommendation

	// Determine disagreement level
	var disagreementLevel string
	if variance != nil {
		if *variance < 1.0 {
			disagreementLevel = "LOW"
		} else if *variance < 2.0 {
			disagreementLevel = "MEDIUM"
		} else {
			disagreementLevel = "HIGH"
		}
	}
	consensus.DisagreementLevel = &disagreementLevel

	// Calculate confidence (0-1)
	if count > 0 {
		confidence := 1.0 - (float64(*variance) / 10.0)
		if confidence < 0 {
			confidence = 0
		}
		consensus.ConfidenceScore = &confidence
	}

	now := time.Now()
	consensus.CalculatedAt = &now
	consensus.CreatedAt = now
	consensus.UpdatedAt = now

	// Upsert consensus
	insertQuery := `
		INSERT INTO interview_consensus (
			session_id, avg_technical_score, avg_communication_score,
			avg_problem_solving_score, avg_culture_fit_score, avg_coding_quality_score,
			score_variance, disagreement_level, consensus_recommendation,
			confidence_score, calculated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (session_id) DO UPDATE SET
			avg_technical_score = $2,
			avg_communication_score = $3,
			avg_problem_solving_score = $4,
			avg_culture_fit_score = $5,
			avg_coding_quality_score = $6,
			score_variance = $7,
			disagreement_level = $8,
			consensus_recommendation = $9,
			confidence_score = $10,
			calculated_at = $11,
			updated_at = NOW()
	`

	_, err = r.db.ExecContext(ctx, insertQuery,
		sessionID,
		consensus.AvgTechnicalScore,
		consensus.AvgCommunicationScore,
		consensus.AvgProblemSolvingScore,
		consensus.AvgCultureFitScore,
		consensus.AvgCodingQualityScore,
		consensus.ScoreVariance,
		consensus.DisagreementLevel,
		consensus.ConsensusRecommendation,
		consensus.ConfidenceScore,
		consensus.CalculatedAt,
	)

	if err != nil {
		return nil, err
	}

	return consensus, nil
}

// GetConsensus retrieves consensus for session
func (r *PostgresRepository) GetConsensus(ctx context.Context, sessionID uuid.UUID) (*domain.InterviewConsensus, error) {
	query := `
		SELECT id, session_id, avg_technical_score, avg_communication_score,
			avg_problem_solving_score, avg_culture_fit_score, avg_coding_quality_score,
			score_variance, disagreement_level, consensus_recommendation,
			confidence_score, calculated_at, created_at, updated_at
		FROM interview_consensus
		WHERE session_id = $1
	`

	consensus := &domain.InterviewConsensus{}
	err := r.db.QueryRowContext(ctx, query, sessionID).Scan(
		&consensus.ID,
		&consensus.SessionID,
		&consensus.AvgTechnicalScore,
		&consensus.AvgCommunicationScore,
		&consensus.AvgProblemSolvingScore,
		&consensus.AvgCultureFitScore,
		&consensus.AvgCodingQualityScore,
		&consensus.ScoreVariance,
		&consensus.DisagreementLevel,
		&consensus.ConsensusRecommendation,
		&consensus.ConfidenceScore,
		&consensus.CalculatedAt,
		&consensus.CreatedAt,
		&consensus.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return consensus, nil
}
