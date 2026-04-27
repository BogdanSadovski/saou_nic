package domain

import (
	"time"
)

// ResumeStatus represents the processing state of a resume
type ResumeStatus string

const (
	StatusPending    ResumeStatus = "pending"
	StatusProcessing ResumeStatus = "processing"
	StatusCompleted  ResumeStatus = "completed"
	StatusFailed     ResumeStatus = "failed"
)

// Resume represents a parsed resume document
type Resume struct {
	ID              string                 `json:"id"`
	UserID          string                 `json:"user_id"`
	FileName        string                 `json:"file_name"`
	FileURL         string                 `json:"file_url,omitempty"`
	ContentType     string                 `json:"content_type"`
	Status          ResumeStatus           `json:"status"`
	FirstName       string                 `json:"first_name,omitempty"`
	LastName        string                `json:"last_name,omitempty"`
	Email           string                 `json:"email,omitempty"`
	Phone           string                 `json:"phone,omitempty"`
	Summary         string                 `json:"summary,omitempty"`
	Skills          []string               `json:"skills,omitempty"`
	Experience      []ExperienceEntry      `json:"experience,omitempty"`
	Education       []EducationEntry       `json:"education,omitempty"`
	Languages       []string               `json:"languages,omitempty"`
	Certifications  []string               `json:"certifications,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	Error           string                 `json:"error,omitempty"`
}

// ExperienceEntry represents a work experience entry
type ExperienceEntry struct {
	Company     string `json:"company"`
	Position    string `json:"position"`
	Description string `json:"description,omitempty"`
	StartDate   string `json:"start_date,omitempty"`
	EndDate     string `json:"end_date,omitempty"`
	Current     bool   `json:"current"`
}

// EducationEntry represents an education entry
type EducationEntry struct {
	Institution string `json:"institution"`
	Degree      string `json:"degree,omitempty"`
	Field       string `json:"field,omitempty"`
	StartDate   string `json:"start_date,omitempty"`
	EndDate     string `json:"end_date,omitempty"`
}

// CreateResumeInput contains the data needed to create a new resume
type CreateResumeInput struct {
	UserID      string
	FileName    string
	ContentType string
	FileData    []byte
}

// UpdateResumeInput contains the data needed to update a resume
type UpdateResumeInput struct {
	ID     string
	UserID string
	// Fields to update
	FirstName      *string
	LastName       *string
	Email          *string
	Phone          *string
	Summary        *string
	Skills         *[]string
	Experience     *[]ExperienceEntry
	Education      *[]EducationEntry
	Languages      *[]string
	Certifications *[]string
	Status         *ResumeStatus
}

// ResumeFilter contains filter options for listing resumes
type ResumeFilter struct {
	UserID     *string
	Status     *ResumeStatus
	CreatedAfter *time.Time
	CreatedBefore *time.Time
	Limit      int
	Offset     int
}
