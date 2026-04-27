package repository

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/bogdan/real_ass/report-service/internal/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Storage implements domain.FileStorage using S3-compatible storage
type S3Storage struct {
	client *minio.Client
	bucket string
}

// NewS3Storage creates a new S3 storage client
func NewS3Storage(cfg config.S3Config) (*S3Storage, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("creating S3 client: %w", err)
	}

	// Ensure bucket exists
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("checking bucket existence: %w", err)
	}

	if !exists {
		if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{Region: cfg.Region}); err != nil {
			return nil, fmt.Errorf("creating bucket: %w", err)
		}
	}

	return &S3Storage{
		client: client,
		bucket: cfg.Bucket,
	}, nil
}

// Upload stores a file and returns its URL
func (s *S3Storage) Upload(ctx context.Context, fileName string, contentType string, data []byte) (string, error) {
	reader := bytes.NewReader(data)
	objectName := generateObjectName(fileName)

	_, err := s.client.PutObject(ctx, s.bucket, objectName, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("uploading to S3: %w", err)
	}

	// Generate a presigned URL valid for 7 days
	url, err := s.GeneratePresignedURL(ctx, objectName, 7*24*3600)
	if err != nil {
		return "", fmt.Errorf("generating presigned URL: %w", err)
	}

	return url, nil
}

// Download retrieves a file by its URL
func (s *S3Storage) Download(ctx context.Context, url string) ([]byte, error) {
	objectName := extractObjectName(url)

	obj, err := s.client.GetObject(ctx, s.bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("downloading from S3: %w", err)
	}
	defer obj.Close()

	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, fmt.Errorf("reading S3 object: %w", err)
	}

	return data, nil
}

// Delete removes a file from storage
func (s *S3Storage) Delete(ctx context.Context, url string) error {
	objectName := extractObjectName(url)

	if err := s.client.RemoveObject(ctx, s.bucket, objectName, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("deleting from S3: %w", err)
	}

	return nil
}

// Exists checks if a file exists in storage
func (s *S3Storage) Exists(ctx context.Context, url string) (bool, error) {
	objectName := extractObjectName(url)

	_, err := s.client.StatObject(ctx, s.bucket, objectName, minio.StatObjectOptions{})
	if err != nil {
		errResponse := minio.ToErrorResponse(err)
		if errResponse.Code == "NoSuchKey" {
			return false, nil
		}
		return false, fmt.Errorf("checking S3 object existence: %w", err)
	}

	return true, nil
}

// GeneratePresignedURL creates a time-limited access URL
func (s *S3Storage) GeneratePresignedURL(ctx context.Context, key string, expiresInSeconds int) (string, error) {
	reqParams := url.Values{}

	url, err := s.client.PresignedGetObject(ctx, s.bucket, key, time.Duration(expiresInSeconds)*time.Second, reqParams)
	if err != nil {
		return "", fmt.Errorf("generating presigned URL: %w", err)
	}

	return url.String(), nil
}

// generateObjectName creates a unique object name with timestamp prefix
func generateObjectName(fileName string) string {
	timestamp := time.Now().Format("2006/01/02")
	return fmt.Sprintf("reports/%s/%s", timestamp, fileName)
}

// extractObjectName extracts the object key from a presigned URL
func extractObjectName(url string) string {
	// For simplicity, this assumes the URL contains the object name directly
	// In production, you'd store the mapping in the database
	return url
}
