package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type CodeSubmission struct {
	ID                 uuid.UUID `json:"id"`
	SessionID          uuid.UUID `json:"session_id"`
	UserID             uuid.UUID `json:"user_id"`
	Language           string    `json:"language"`
	Code               string    `json:"code"`
	InputData          *string   `json:"input_data,omitempty"`
	SubmissionSequence int       `json:"submission_sequence"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type CodeExecutionResult struct {
	ID              uuid.UUID       `json:"id"`
	SubmissionID    uuid.UUID       `json:"submission_id"`
	Status          string          `json:"status"`
	Output          *string         `json:"output,omitempty"`
	ErrorMessage    *string         `json:"error_message,omitempty"`
	ExecutionTimeMs *int64          `json:"execution_time_ms,omitempty"`
	MemoryUsedBytes *int64          `json:"memory_used_bytes,omitempty"`
	ExitCode        *int            `json:"exit_code,omitempty"`
	TestResults     json.RawMessage `json:"test_results,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
}

type CodeTestCase struct {
	ID             uuid.UUID `json:"id"`
	QuestionID     uuid.UUID `json:"question_id"`
	TestName       string    `json:"test_name"`
	InputData      string    `json:"input_data"`
	ExpectedOutput string    `json:"expected_output"`
	IsHidden       bool      `json:"is_hidden"`
	Sequence       int       `json:"sequence"`
	CreatedAt      time.Time `json:"created_at"`
}

func (r *PostgresRepository) CreateCodeSubmission(ctx context.Context, submission *CodeSubmission) (*CodeSubmission, error) {
	query := `
		INSERT INTO code_submissions (session_id, user_id, language, code, input_data, submission_sequence)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, session_id, user_id, language, code, input_data, submission_sequence, created_at, updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		submission.SessionID,
		submission.UserID,
		submission.Language,
		submission.Code,
		submission.InputData,
		submission.SubmissionSequence,
	).Scan(
		&submission.ID,
		&submission.SessionID,
		&submission.UserID,
		&submission.Language,
		&submission.Code,
		&submission.InputData,
		&submission.SubmissionSequence,
		&submission.CreatedAt,
		&submission.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return submission, nil
}

func (r *PostgresRepository) CreateCodeExecutionResult(ctx context.Context, result *CodeExecutionResult) (*CodeExecutionResult, error) {
	query := `
		INSERT INTO code_execution_results (submission_id, status, output, error_message, execution_time_ms, memory_used_bytes, exit_code, test_results)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, submission_id, status, output, error_message, execution_time_ms, memory_used_bytes, exit_code, test_results, created_at
	`

	err := r.db.QueryRowContext(ctx, query,
		result.SubmissionID,
		result.Status,
		result.Output,
		result.ErrorMessage,
		result.ExecutionTimeMs,
		result.MemoryUsedBytes,
		result.ExitCode,
		result.TestResults,
	).Scan(
		&result.ID,
		&result.SubmissionID,
		&result.Status,
		&result.Output,
		&result.ErrorMessage,
		&result.ExecutionTimeMs,
		&result.MemoryUsedBytes,
		&result.ExitCode,
		&result.TestResults,
		&result.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *PostgresRepository) GetCodeExecutionResult(ctx context.Context, submissionID uuid.UUID) (*CodeExecutionResult, error) {
	query := `
		SELECT id, submission_id, status, output, error_message, execution_time_ms, memory_used_bytes, exit_code, test_results, created_at
		FROM code_execution_results
		WHERE submission_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	result := &CodeExecutionResult{}
	err := r.db.QueryRowContext(ctx, query, submissionID).Scan(
		&result.ID,
		&result.SubmissionID,
		&result.Status,
		&result.Output,
		&result.ErrorMessage,
		&result.ExecutionTimeMs,
		&result.MemoryUsedBytes,
		&result.ExitCode,
		&result.TestResults,
		&result.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *PostgresRepository) ListCodeSubmissionsBySession(ctx context.Context, sessionID uuid.UUID) ([]*CodeSubmission, error) {
	query := `
		SELECT id, session_id, user_id, language, code, input_data, submission_sequence, created_at, updated_at
		FROM code_submissions
		WHERE session_id = $1
		ORDER BY submission_sequence ASC
	`

	rows, err := r.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var submissions []*CodeSubmission
	for rows.Next() {
		sub := &CodeSubmission{}
		err := rows.Scan(
			&sub.ID,
			&sub.SessionID,
			&sub.UserID,
			&sub.Language,
			&sub.Code,
			&sub.InputData,
			&sub.SubmissionSequence,
			&sub.CreatedAt,
			&sub.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		submissions = append(submissions, sub)
	}

	return submissions, rows.Err()
}

func (r *PostgresRepository) CreateCodeTestCase(ctx context.Context, testCase *CodeTestCase) (*CodeTestCase, error) {
	query := `
		INSERT INTO code_test_cases (question_id, test_name, input_data, expected_output, is_hidden, sequence)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, question_id, test_name, input_data, expected_output, is_hidden, sequence, created_at
	`

	err := r.db.QueryRowContext(ctx, query,
		testCase.QuestionID,
		testCase.TestName,
		testCase.InputData,
		testCase.ExpectedOutput,
		testCase.IsHidden,
		testCase.Sequence,
	).Scan(
		&testCase.ID,
		&testCase.QuestionID,
		&testCase.TestName,
		&testCase.InputData,
		&testCase.ExpectedOutput,
		&testCase.IsHidden,
		&testCase.Sequence,
		&testCase.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return testCase, nil
}

func (r *PostgresRepository) ListCodeTestCasesByQuestion(ctx context.Context, questionID uuid.UUID) ([]*CodeTestCase, error) {
	query := `
		SELECT id, question_id, test_name, input_data, expected_output, is_hidden, sequence, created_at
		FROM code_test_cases
		WHERE question_id = $1
		ORDER BY sequence ASC
	`

	rows, err := r.db.QueryContext(ctx, query, questionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var testCases []*CodeTestCase
	for rows.Next() {
		tc := &CodeTestCase{}
		err := rows.Scan(
			&tc.ID,
			&tc.QuestionID,
			&tc.TestName,
			&tc.InputData,
			&tc.ExpectedOutput,
			&tc.IsHidden,
			&tc.Sequence,
			&tc.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		testCases = append(testCases, tc)
	}

	return testCases, rows.Err()
}
