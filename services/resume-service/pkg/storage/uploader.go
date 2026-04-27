package storage

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// Uploader handles file uploads with validation
type Uploader struct {
	allowedTypes map[string]bool
	maxFileSize  int64
}

// UploadResult contains the result of a file upload
type UploadResult struct {
	FileID      string
	FileName    string
	ContentType string
	Size        int64
	Key         string
}

// UploaderOption configures the Uploader
type UploaderOption func(*Uploader)

// NewUploader creates a new file uploader
func NewUploader(opts ...UploaderOption) *Uploader {
	uploader := &Uploader{
		allowedTypes: map[string]bool{
			"application/pdf": true,
			"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
			"application/msword": true,
			"text/plain": true,
			"application/rtf": true,
		},
		maxFileSize: 10 * 1024 * 1024, // 10MB
	}

	for _, opt := range opts {
		opt(uploader)
	}

	return uploader
}

// WithAllowedTypes sets the allowed MIME types
func WithAllowedTypes(types []string) UploaderOption {
	return func(u *Uploader) {
		u.allowedTypes = make(map[string]bool)
		for _, t := range types {
			u.allowedTypes[t] = true
		}
	}
}

// WithMaxFileSize sets the maximum file size
func WithMaxFileSize(size int64) UploaderOption {
	return func(u *Uploader) {
		u.maxFileSize = size
	}
}

// Upload processes a multipart file upload
func (u *Uploader) Upload(ctx context.Context, fileHeader *multipart.FileHeader) (*UploadResult, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Validate file size
	if fileHeader.Size > u.maxFileSize {
		return nil, fmt.Errorf("file size %d exceeds maximum allowed size %d", fileHeader.Size, u.maxFileSize)
	}

	// Read file content
	data := make([]byte, fileHeader.Size)
	if _, err := io.ReadFull(file, data); err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Detect content type
	contentType := detectContentType(data, fileHeader.Filename)

	// Validate content type
	if !u.allowedTypes[contentType] {
		return nil, fmt.Errorf("file type %s is not allowed", contentType)
	}

	// Generate unique file ID
	fileID := uuid.New().String()
	ext := filepath.Ext(fileHeader.Filename)
	key := fmt.Sprintf("%s%s", fileID, ext)

	return &UploadResult{
		FileID:      fileID,
		FileName:    fileHeader.Filename,
		ContentType: contentType,
		Size:        fileHeader.Size,
		Key:         key,
	}, nil
}

// UploadFromReader uploads data from an io.Reader
func (u *Uploader) UploadFromReader(ctx context.Context, reader io.Reader, fileName string, size int64) (*UploadResult, error) {
	// Validate file size
	if size > u.maxFileSize {
		return nil, fmt.Errorf("file size %d exceeds maximum allowed size %d", size, u.maxFileSize)
	}

	// Read file content
	data := make([]byte, size)
	if _, err := io.ReadFull(reader, data); err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Detect content type
	contentType := detectContentType(data, fileName)

	// Validate content type
	if !u.allowedTypes[contentType] {
		return nil, fmt.Errorf("file type %s is not allowed", contentType)
	}

	// Generate unique file ID
	fileID := uuid.New().String()
	ext := filepath.Ext(fileName)
	key := fmt.Sprintf("%s%s", fileID, ext)

	return &UploadResult{
		FileID:      fileID,
		FileName:    fileName,
		ContentType: contentType,
		Size:        size,
		Key:         key,
	}, nil
}

// ValidateFile checks if a file meets the upload requirements
func (u *Uploader) ValidateFile(fileName string, contentType string, size int64) error {
	if fileName == "" {
		return fmt.Errorf("file name is required")
	}

	if contentType == "" {
		return fmt.Errorf("content type is required")
	}

	if !u.allowedTypes[contentType] {
		return fmt.Errorf("file type %s is not allowed", contentType)
	}

	if size <= 0 {
		return fmt.Errorf("invalid file size: %d", size)
	}

	if size > u.maxFileSize {
		return fmt.Errorf("file size %d exceeds maximum allowed size %d", size, u.maxFileSize)
	}

	return nil
}

// GetAllowedTypes returns the list of allowed MIME types
func (u *Uploader) GetAllowedTypes() []string {
	var types []string
	for t := range u.allowedTypes {
		types = append(types, t)
	}
	return types
}

// detectContentType determines the MIME type of file data
func detectContentType(data []byte, fileName string) string {
	// Use http.DetectContentType for basic detection
	// For more accurate detection, use a library like gabriel-vasile/mimetype

	// Check by extension first
	ext := strings.ToLower(filepath.Ext(fileName))
	extToMime := map[string]string{
		".pdf":  "application/pdf",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".doc":  "application/msword",
		".txt":  "text/plain",
		".rtf":  "application/rtf",
	}

	if mime, ok := extToMime[ext]; ok {
		return mime
	}

	// Fallback to content-based detection
	// In production, use a proper MIME type detection library
	return "application/octet-stream"
}
