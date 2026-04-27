package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/interview-platform/interview-service/internal/domain"
	"github.com/lib/pq"
)

func (r *PostgresRepository) CreateInterviewModuleSession(ctx context.Context, session *domain.InterviewModuleSession) error {
	if session.ID == uuid.Nil {
		session.ID = generateID()
	}

	if session.StartedAt.IsZero() {
		session.StartedAt = time.Now()
	}

	metadataJSON, err := json.Marshal(session.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO interview_sessions (
			id, user_id, role, level, status, current_topic, difficulty_score,
			pressure_level, question_count, question_limit, started_at, ended_at,
			duration_seconds, metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12,
			$13, $14, NOW(), NOW()
		)
	`

	_, err = r.db.ExecContext(
		ctx,
		query,
		session.ID,
		session.UserID,
		session.Role,
		session.Level,
		session.Status,
		session.CurrentTopic,
		session.DifficultyScore,
		session.PressureLevel,
		session.QuestionCount,
		session.QuestionLimit,
		session.StartedAt,
		session.EndedAt,
		session.DurationSeconds,
		metadataJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to create interview module session: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetInterviewModuleSessionByID(ctx context.Context, id uuid.UUID) (*domain.InterviewModuleSession, error) {
	query := `
		SELECT id, user_id, role, level, status, current_topic, difficulty_score,
		       pressure_level, question_count, question_limit, started_at, ended_at,
		       duration_seconds, metadata, created_at, updated_at
		FROM interview_sessions
		WHERE id = $1
	`

	var s domain.InterviewModuleSession
	var metadataRaw []byte
	if err := r.db.QueryRowContext(ctx, query, id).Scan(
		&s.ID,
		&s.UserID,
		&s.Role,
		&s.Level,
		&s.Status,
		&s.CurrentTopic,
		&s.DifficultyScore,
		&s.PressureLevel,
		&s.QuestionCount,
		&s.QuestionLimit,
		&s.StartedAt,
		&s.EndedAt,
		&s.DurationSeconds,
		&metadataRaw,
		&s.CreatedAt,
		&s.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("failed to get interview module session: %w", err)
	}

	s.Metadata = map[string]interface{}{}
	if len(metadataRaw) > 0 {
		if err := json.Unmarshal(metadataRaw, &s.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal session metadata: %w", err)
		}
	}

	return &s, nil
}

func (r *PostgresRepository) ListInterviewModuleSessionsByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.InterviewModuleSession, error) {
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT id, user_id, role, level, status, current_topic, difficulty_score,
		       pressure_level, question_count, question_limit, started_at, ended_at,
		       duration_seconds, metadata, created_at, updated_at
		FROM interview_sessions
		WHERE user_id = $1
		ORDER BY started_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list interview module sessions by user: %w", err)
	}
	defer rows.Close()

	sessions := make([]*domain.InterviewModuleSession, 0)
	for rows.Next() {
		var s domain.InterviewModuleSession
		var metadataRaw []byte
		if err := rows.Scan(
			&s.ID,
			&s.UserID,
			&s.Role,
			&s.Level,
			&s.Status,
			&s.CurrentTopic,
			&s.DifficultyScore,
			&s.PressureLevel,
			&s.QuestionCount,
			&s.QuestionLimit,
			&s.StartedAt,
			&s.EndedAt,
			&s.DurationSeconds,
			&metadataRaw,
			&s.CreatedAt,
			&s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan interview module session: %w", err)
		}

		s.Metadata = map[string]interface{}{}
		if len(metadataRaw) > 0 {
			if err := json.Unmarshal(metadataRaw, &s.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal interview module session metadata: %w", err)
			}
		}

		sessions = append(sessions, &s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate interview module sessions: %w", err)
	}

	return sessions, nil
}

func (r *PostgresRepository) UpdateInterviewModuleSession(ctx context.Context, session *domain.InterviewModuleSession) error {
	metadataJSON, err := json.Marshal(session.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		UPDATE interview_sessions
		SET status = $1,
		    current_topic = $2,
		    difficulty_score = $3,
		    pressure_level = $4,
		    question_count = $5,
		    question_limit = $6,
		    ended_at = $7,
		    duration_seconds = $8,
		    metadata = $9,
		    updated_at = NOW()
		WHERE id = $10
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		session.Status,
		session.CurrentTopic,
		session.DifficultyScore,
		session.PressureLevel,
		session.QuestionCount,
		session.QuestionLimit,
		session.EndedAt,
		session.DurationSeconds,
		metadataJSON,
		session.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update interview module session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows for session update: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("interview module session not found: %s", session.ID)
	}

	return nil
}

func (r *PostgresRepository) CreateInterviewModuleMessage(ctx context.Context, message *domain.InterviewModuleMessage) error {
	if message.ID == uuid.Nil {
		message.ID = generateID()
	}

	tokenUsageJSON, err := json.Marshal(message.TokenUsage)
	if err != nil {
		return fmt.Errorf("failed to marshal token usage: %w", err)
	}

	query := `
		INSERT INTO interview_messages (
			id, session_id, sender, content, topic, difficulty, created_at, token_usage
		) VALUES (
			$1, $2, $3, $4, $5, $6, COALESCE($7, NOW()), $8
		)
	`

	_, err = r.db.ExecContext(
		ctx,
		query,
		message.ID,
		message.SessionID,
		message.Sender,
		message.Content,
		message.Topic,
		message.Difficulty,
		message.CreatedAt,
		nullableJSON(tokenUsageJSON),
	)
	if err != nil {
		return fmt.Errorf("failed to create interview module message: %w", err)
	}

	return nil
}

func (r *PostgresRepository) ListInterviewModuleMessagesBySessionID(ctx context.Context, sessionID uuid.UUID, limit, offset int) ([]*domain.InterviewModuleMessage, error) {
	query := `
		SELECT id, session_id, sender, content, topic, difficulty, created_at, token_usage
		FROM interview_messages
		WHERE session_id = $1
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, sessionID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list interview module messages: %w", err)
	}
	defer rows.Close()

	messages := make([]*domain.InterviewModuleMessage, 0)
	for rows.Next() {
		var m domain.InterviewModuleMessage
		var tokenUsageRaw []byte
		if err := rows.Scan(
			&m.ID,
			&m.SessionID,
			&m.Sender,
			&m.Content,
			&m.Topic,
			&m.Difficulty,
			&m.CreatedAt,
			&tokenUsageRaw,
		); err != nil {
			return nil, fmt.Errorf("failed to scan interview module message: %w", err)
		}

		m.TokenUsage = map[string]interface{}{}
		if len(tokenUsageRaw) > 0 {
			if err := json.Unmarshal(tokenUsageRaw, &m.TokenUsage); err != nil {
				return nil, fmt.Errorf("failed to unmarshal token usage: %w", err)
			}
		}
		messages = append(messages, &m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate interview module messages: %w", err)
	}

	return messages, nil
}

func (r *PostgresRepository) UpsertInterviewModuleReport(ctx context.Context, report *domain.InterviewModuleReport) error {
	if report.ID == uuid.Nil {
		report.ID = generateID()
	}

	strengths, err := marshalJSONArray(report.Strengths)
	if err != nil {
		return fmt.Errorf("failed to marshal strengths: %w", err)
	}
	weaknesses, err := marshalJSONArray(report.Weaknesses)
	if err != nil {
		return fmt.Errorf("failed to marshal weaknesses: %w", err)
	}
	recommendations, err := marshalJSONArray(report.Recommendations)
	if err != nil {
		return fmt.Errorf("failed to marshal recommendations: %w", err)
	}

	query := `
		INSERT INTO interview_reports (
			id, session_id, correctness, clarity, completeness, relevance,
			overall_score, strengths, weaknesses, recommendations, generated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, COALESCE($11, NOW())
		)
		ON CONFLICT (session_id) DO UPDATE
		SET correctness = EXCLUDED.correctness,
		    clarity = EXCLUDED.clarity,
		    completeness = EXCLUDED.completeness,
		    relevance = EXCLUDED.relevance,
		    overall_score = EXCLUDED.overall_score,
		    strengths = EXCLUDED.strengths,
		    weaknesses = EXCLUDED.weaknesses,
		    recommendations = EXCLUDED.recommendations,
		    generated_at = EXCLUDED.generated_at
	`

	_, err = r.db.ExecContext(
		ctx,
		query,
		report.ID,
		report.SessionID,
		report.Correctness,
		report.Clarity,
		report.Completeness,
		report.Relevance,
		report.OverallScore,
		pq.Array(strengths),
		pq.Array(weaknesses),
		pq.Array(recommendations),
		report.GeneratedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert interview module report: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetInterviewModuleReportBySessionID(ctx context.Context, sessionID uuid.UUID) (*domain.InterviewModuleReport, error) {
	query := `
		SELECT id, session_id, correctness, clarity, completeness, relevance,
		       overall_score, strengths, weaknesses, recommendations, generated_at
		FROM interview_reports
		WHERE session_id = $1
	`

	var report domain.InterviewModuleReport
	var strengthsRaw []string
	var weaknessesRaw []string
	var recommendationsRaw []string
	if err := r.db.QueryRowContext(ctx, query, sessionID).Scan(
		&report.ID,
		&report.SessionID,
		&report.Correctness,
		&report.Clarity,
		&report.Completeness,
		&report.Relevance,
		&report.OverallScore,
		pq.Array(&strengthsRaw),
		pq.Array(&weaknessesRaw),
		pq.Array(&recommendationsRaw),
		&report.GeneratedAt,
	); err != nil {
		return nil, fmt.Errorf("failed to get interview module report: %w", err)
	}

	var err error
	report.Strengths, err = unmarshalJSONArray(strengthsRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse strengths: %w", err)
	}
	report.Weaknesses, err = unmarshalJSONArray(weaknessesRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse weaknesses: %w", err)
	}
	report.Recommendations, err = unmarshalJSONArray(recommendationsRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse recommendations: %w", err)
	}

	return &report, nil
}

func (r *PostgresRepository) UpsertRequestLog(ctx context.Context, log *domain.RequestLog) error {
	query := `
		INSERT INTO request_log (idempotency_key, session_id, response_hash, created_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (idempotency_key) DO UPDATE
		SET session_id = EXCLUDED.session_id,
		    response_hash = EXCLUDED.response_hash
	`

	_, err := r.db.ExecContext(ctx, query, log.IdempotencyKey, nullableUUID(log.SessionID), log.ResponseHash)
	if err != nil {
		return fmt.Errorf("failed to upsert request log: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetRequestLogByKey(ctx context.Context, idempotencyKey string) (*domain.RequestLog, error) {
	query := `
		SELECT idempotency_key, session_id, response_hash, created_at
		FROM request_log
		WHERE idempotency_key = $1
	`

	var log domain.RequestLog
	var sessionID sql.NullString
	if err := r.db.QueryRowContext(ctx, query, idempotencyKey).Scan(
		&log.IdempotencyKey,
		&sessionID,
		&log.ResponseHash,
		&log.CreatedAt,
	); err != nil {
		return nil, fmt.Errorf("failed to get request log: %w", err)
	}

	if sessionID.Valid && strings.TrimSpace(sessionID.String) != "" {
		parsed, err := uuid.Parse(sessionID.String)
		if err != nil {
			return nil, fmt.Errorf("failed to parse request log session_id: %w", err)
		}
		log.SessionID = parsed
	}

	return &log, nil
}

func nullableJSON(raw []byte) interface{} {
	if len(raw) == 0 || string(raw) == "null" || string(raw) == "{}" {
		return nil
	}
	return raw
}

func nullableUUID(id uuid.UUID) interface{} {
	if id == uuid.Nil {
		return nil
	}
	return id
}

func marshalJSONArray(items []map[string]interface{}) ([]string, error) {
	result := make([]string, 0, len(items))
	for _, item := range items {
		raw, err := json.Marshal(item)
		if err != nil {
			return nil, err
		}
		result = append(result, string(raw))
	}
	return result, nil
}

func unmarshalJSONArray(items []string) ([]map[string]interface{}, error) {
	result := make([]map[string]interface{}, 0, len(items))
	for _, raw := range items {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		parsed := map[string]interface{}{}
		if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
			return nil, err
		}
		result = append(result, parsed)
	}
	return result, nil
}
