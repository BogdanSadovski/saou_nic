package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/interview-platform/interview-service/internal/domain"

	"github.com/google/uuid"
)

// ==================== Interview Repository Implementation ====================

func (r *PostgresRepository) CreateInterview(ctx context.Context, interview *domain.Interview) error {
	if interview.ID == uuid.Nil {
		interview.ID = generateID()
	}

	query := `
		INSERT INTO interviews (
			id, interviewer_id, candidate_id, title, description, status,
			scheduled_at, duration, language, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
	`

	_, err := r.db.ExecContext(ctx, query,
		interview.ID,
		interview.InterviewerID,
		interview.CandidateID,
		interview.Title,
		interview.Description,
		interview.Status,
		interview.ScheduledAt,
		interview.Duration,
		interview.Language,
	)
	if err != nil {
		return fmt.Errorf("failed to create interview: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetInterviewByID(ctx context.Context, id uuid.UUID) (*domain.Interview, error) {
	query := `
		SELECT id, interviewer_id, candidate_id, title, description, status,
		       scheduled_at, duration, language, created_at, updated_at
		FROM interviews
		WHERE id = $1
	`

	return scanInterview(r.db.QueryRowContext(ctx, query, id))
}

func (r *PostgresRepository) GetInterviewsByInterviewerID(ctx context.Context, interviewerID uuid.UUID, limit, offset int) ([]*domain.Interview, error) {
	query := `
		SELECT id, interviewer_id, candidate_id, title, description, status,
		       scheduled_at, duration, language, created_at, updated_at
		FROM interviews
		WHERE interviewer_id = $1
		ORDER BY scheduled_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, interviewerID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query interviews: %w", err)
	}

	return scanInterviews(rows)
}

func (r *PostgresRepository) GetInterviewsByCandidateID(ctx context.Context, candidateID uuid.UUID, limit, offset int) ([]*domain.Interview, error) {
	query := `
		SELECT id, interviewer_id, candidate_id, title, description, status,
		       scheduled_at, duration, language, created_at, updated_at
		FROM interviews
		WHERE candidate_id = $1
		ORDER BY scheduled_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, candidateID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query interviews: %w", err)
	}

	return scanInterviews(rows)
}

func (r *PostgresRepository) UpdateInterview(ctx context.Context, interview *domain.Interview) error {
	query := `
		UPDATE interviews
		SET title = $1, description = $2, status = $3, duration = $4,
		    language = $5, updated_at = NOW()
		WHERE id = $6
	`

	result, err := r.db.ExecContext(ctx, query,
		interview.Title,
		interview.Description,
		interview.Status,
		interview.Duration,
		interview.Language,
		interview.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update interview: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("interview not found: %s", interview.ID)
	}

	return nil
}

func (r *PostgresRepository) DeleteInterview(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM interviews WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete interview: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("interview not found: %s", id)
	}

	return nil
}

func (r *PostgresRepository) ListInterviews(ctx context.Context, status *domain.InterviewStatus, limit, offset int) ([]*domain.Interview, error) {
	var rows *sql.Rows
	var err error

	if status != nil {
		query := `
			SELECT id, interviewer_id, candidate_id, title, description, status,
			       scheduled_at, duration, language, created_at, updated_at
			FROM interviews
			WHERE status = $1
			ORDER BY scheduled_at DESC
			LIMIT $2 OFFSET $3
		`
		rows, err = r.db.QueryContext(ctx, query, *status, limit, offset)
	} else {
		query := `
			SELECT id, interviewer_id, candidate_id, title, description, status,
			       scheduled_at, duration, language, created_at, updated_at
			FROM interviews
			ORDER BY scheduled_at DESC
			LIMIT $1 OFFSET $2
		`
		rows, err = r.db.QueryContext(ctx, query, limit, offset)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list interviews: %w", err)
	}

	return scanInterviews(rows)
}

// ==================== Question Repository Implementation ====================

func (r *PostgresRepository) CreateQuestion(ctx context.Context, question *domain.Question) error {
	if question.ID == uuid.Nil {
		question.ID = generateID()
	}

	tagsJSON, _ := json.Marshal(question.Tags)
	testCasesJSON, _ := json.Marshal(question.TestCases)

	query := `
		INSERT INTO questions (
			id, interview_id, title, description, type, difficulty, tags,
			starter_code, solution, test_cases, points, question_order, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
	`

	_, err := r.db.ExecContext(ctx, query,
		question.ID,
		question.InterviewID,
		question.Title,
		question.Description,
		question.Type,
		question.Difficulty,
		tagsJSON,
		question.StarterCode,
		question.Solution,
		testCasesJSON,
		question.Points,
		question.Order,
	)
	if err != nil {
		return fmt.Errorf("failed to create question: %w", err)
	}

	return nil
}

func (r *PostgresRepository) CreateQuestionsBatch(ctx context.Context, questions []*domain.Question) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, q := range questions {
		if q.ID == uuid.Nil {
			q.ID = generateID()
		}

		tagsJSON, _ := json.Marshal(q.Tags)
		testCasesJSON, _ := json.Marshal(q.TestCases)

		query := `
			INSERT INTO questions (
				id, interview_id, title, description, type, difficulty, tags,
				starter_code, solution, test_cases, points, question_order, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
		`

		_, err := tx.ExecContext(ctx, query,
			q.ID,
			q.InterviewID,
			q.Title,
			q.Description,
			q.Type,
			q.Difficulty,
			tagsJSON,
			q.StarterCode,
			q.Solution,
			testCasesJSON,
			q.Points,
			q.Order,
		)
		if err != nil {
			return fmt.Errorf("failed to create question: %w", err)
		}
	}

	return tx.Commit()
}

func (r *PostgresRepository) GetQuestionByID(ctx context.Context, id uuid.UUID) (*domain.Question, error) {
	query := `
		SELECT id, interview_id, title, description, type, difficulty, tags,
		       starter_code, solution, test_cases, points, question_order, created_at
		FROM questions
		WHERE id = $1
	`

	var q domain.Question
	var tagsJSON, testCasesJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&q.ID,
		&q.InterviewID,
		&q.Title,
		&q.Description,
		&q.Type,
		&q.Difficulty,
		&tagsJSON,
		&q.StarterCode,
		&q.Solution,
		&testCasesJSON,
		&q.Points,
		&q.Order,
		&q.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get question: %w", err)
	}

	if err := json.Unmarshal(tagsJSON, &q.Tags); err != nil {
		return nil, fmt.Errorf("failed to parse tags: %w", err)
	}

	if err := json.Unmarshal(testCasesJSON, &q.TestCases); err != nil {
		return nil, fmt.Errorf("failed to parse test cases: %w", err)
	}

	return &q, nil
}

func (r *PostgresRepository) GetQuestionsByInterviewID(ctx context.Context, interviewID uuid.UUID) ([]*domain.Question, error) {
	query := `
		SELECT id, interview_id, title, description, type, difficulty, tags,
		       starter_code, solution, test_cases, points, question_order, created_at
		FROM questions
		WHERE interview_id = $1
		ORDER BY question_order ASC
	`

	rows, err := r.db.QueryContext(ctx, query, interviewID)
	if err != nil {
		return nil, fmt.Errorf("failed to query questions: %w", err)
	}
	defer rows.Close()

	var questions []*domain.Question
	for rows.Next() {
		var q domain.Question
		var tagsJSON, testCasesJSON []byte

		if err := rows.Scan(
			&q.ID,
			&q.InterviewID,
			&q.Title,
			&q.Description,
			&q.Type,
			&q.Difficulty,
			&tagsJSON,
			&q.StarterCode,
			&q.Solution,
			&testCasesJSON,
			&q.Points,
			&q.Order,
			&q.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan question: %w", err)
		}

		if err := json.Unmarshal(tagsJSON, &q.Tags); err != nil {
			return nil, fmt.Errorf("failed to parse tags: %w", err)
		}

		if err := json.Unmarshal(testCasesJSON, &q.TestCases); err != nil {
			return nil, fmt.Errorf("failed to parse test cases: %w", err)
		}

		questions = append(questions, &q)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return questions, nil
}

func (r *PostgresRepository) UpdateQuestion(ctx context.Context, question *domain.Question) error {
	tagsJSON, _ := json.Marshal(question.Tags)
	testCasesJSON, _ := json.Marshal(question.TestCases)

	query := `
		UPDATE questions
		SET title = $1, description = $2, type = $3, difficulty = $4,
		    tags = $5, starter_code = $6, solution = $7, test_cases = $8,
		    points = $9, question_order = $10
		WHERE id = $11
	`

	result, err := r.db.ExecContext(ctx, query,
		question.Title,
		question.Description,
		question.Type,
		question.Difficulty,
		tagsJSON,
		question.StarterCode,
		question.Solution,
		testCasesJSON,
		question.Points,
		question.Order,
		question.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update question: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("question not found: %s", question.ID)
	}

	return nil
}

func (r *PostgresRepository) DeleteQuestion(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM questions WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete question: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("question not found: %s", id)
	}

	return nil
}

// ==================== Session Repository Implementation ====================

func (r *PostgresRepository) CreateSession(ctx context.Context, session *domain.Session) error {
	if session.ID == uuid.Nil {
		session.ID = generateID()
	}

	query := `
		INSERT INTO sessions (
			id, interview_id, status, current_question_index, start_time,
			end_time, score, feedback, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
	`

	_, err := r.db.ExecContext(ctx, query,
		session.ID,
		session.InterviewID,
		session.Status,
		session.CurrentQIndex,
		session.StartTime,
		session.EndTime,
		session.Score,
		session.Feedback,
	)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetSessionByID(ctx context.Context, id uuid.UUID) (*domain.Session, error) {
	query := `
		SELECT id, interview_id, status, current_question_index, start_time,
		       end_time, score, feedback, created_at, updated_at
		FROM sessions
		WHERE id = $1
	`

	var s domain.Session
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&s.ID,
		&s.InterviewID,
		&s.Status,
		&s.CurrentQIndex,
		&s.StartTime,
		&s.EndTime,
		&s.Score,
		&s.Feedback,
		&s.CreatedAt,
		&s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return &s, nil
}

func (r *PostgresRepository) GetSessionByInterviewID(ctx context.Context, interviewID uuid.UUID) (*domain.Session, error) {
	query := `
		SELECT id, interview_id, status, current_question_index, start_time,
		       end_time, score, feedback, created_at, updated_at
		FROM sessions
		WHERE interview_id = $1
	`

	var s domain.Session
	err := r.db.QueryRowContext(ctx, query, interviewID).Scan(
		&s.ID,
		&s.InterviewID,
		&s.Status,
		&s.CurrentQIndex,
		&s.StartTime,
		&s.EndTime,
		&s.Score,
		&s.Feedback,
		&s.CreatedAt,
		&s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return &s, nil
}

func (r *PostgresRepository) UpdateSession(ctx context.Context, session *domain.Session) error {
	query := `
		UPDATE sessions
		SET status = $1, current_question_index = $2, end_time = $3,
		    score = $4, feedback = $5, updated_at = NOW()
		WHERE id = $6
	`

	result, err := r.db.ExecContext(ctx, query,
		session.Status,
		session.CurrentQIndex,
		session.EndTime,
		session.Score,
		session.Feedback,
		session.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("session not found: %s", session.ID)
	}

	return nil
}

func (r *PostgresRepository) DeleteSession(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM sessions WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("session not found: %s", id)
	}

	return nil
}

// ==================== Answer Repository Implementation ====================

func (r *PostgresRepository) CreateAnswer(ctx context.Context, answer *domain.Answer) error {
	if answer.ID == uuid.Nil {
		answer.ID = generateID()
	}

	query := `
		INSERT INTO answers (
			id, session_id, question_id, code, language, is_correct,
			score, submitted_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
	`

	_, err := r.db.ExecContext(ctx, query,
		answer.ID,
		answer.SessionID,
		answer.QuestionID,
		answer.Code,
		answer.Language,
		answer.IsCorrect,
		answer.Score,
		answer.SubmittedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create answer: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetAnswerByID(ctx context.Context, id uuid.UUID) (*domain.Answer, error) {
	query := `
		SELECT id, session_id, question_id, code, language, is_correct,
		       score, submitted_at, created_at
		FROM answers
		WHERE id = $1
	`

	var a domain.Answer
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&a.ID,
		&a.SessionID,
		&a.QuestionID,
		&a.Code,
		&a.Language,
		&a.IsCorrect,
		&a.Score,
		&a.SubmittedAt,
		&a.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get answer: %w", err)
	}

	return &a, nil
}

func (r *PostgresRepository) GetAnswersBySessionID(ctx context.Context, sessionID uuid.UUID) ([]*domain.Answer, error) {
	query := `
		SELECT id, session_id, question_id, code, language, is_correct,
		       score, submitted_at, created_at
		FROM answers
		WHERE session_id = $1
		ORDER BY submitted_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query answers: %w", err)
	}
	defer rows.Close()

	var answers []*domain.Answer
	for rows.Next() {
		var a domain.Answer
		if err := rows.Scan(
			&a.ID,
			&a.SessionID,
			&a.QuestionID,
			&a.Code,
			&a.Language,
			&a.IsCorrect,
			&a.Score,
			&a.SubmittedAt,
			&a.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan answer: %w", err)
		}
		answers = append(answers, &a)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return answers, nil
}

func (r *PostgresRepository) UpdateAnswer(ctx context.Context, answer *domain.Answer) error {
	query := `
		UPDATE answers
		SET code = $1, language = $2, is_correct = $3, score = $4
		WHERE id = $5
	`

	result, err := r.db.ExecContext(ctx, query,
		answer.Code,
		answer.Language,
		answer.IsCorrect,
		answer.Score,
		answer.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update answer: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("answer not found: %s", answer.ID)
	}

	return nil
}
