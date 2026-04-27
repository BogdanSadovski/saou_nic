package domain

import (
	"context"
)

// ResumeRepository defines the interface for resume data access
type ResumeRepository interface {
	// Create inserts a new resume record
	Create(ctx context.Context, resume *Resume) error

	// GetByID retrieves a resume by its ID
	GetByID(ctx context.Context, id string) (*Resume, error)

	// GetByUserID retrieves all resumes for a specific user
	GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*Resume, error)

	// Update modifies an existing resume
	Update(ctx context.Context, resume *Resume) error

	// Delete removes a resume by its ID
	Delete(ctx context.Context, id string) error

	// List retrieves resumes with optional filtering
	List(ctx context.Context, filter *ResumeFilter) ([]*Resume, int, error)

	// UpdateStatus updates only the status and error fields
	UpdateStatus(ctx context.Context, id string, status ResumeStatus, errMsg string) error

	// Ping checks if the repository is accessible
	Ping(ctx context.Context) error
}

// FileStorage defines the interface for file storage operations
type FileStorage interface {
	// Upload stores a file and returns its URL
	Upload(ctx context.Context, key string, data []byte, contentType string) (string, error)

	// Download retrieves a file's contents by its key
	Download(ctx context.Context, key string) ([]byte, error)

	// Delete removes a file by its key
	Delete(ctx context.Context, key string) error

	// GetURL generates a presigned URL for a file
	GetURL(ctx context.Context, key string, expiresIn int) (string, error)
}
