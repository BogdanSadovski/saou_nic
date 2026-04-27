package repository

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"resume-service/internal/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Storage implements domain.FileStorage
type S3Storage struct {
	client *s3.Client
	bucket string
}

// NewS3Storage creates a new S3 storage client
func NewS3Storage(cfg config.S3Config) (*S3Storage, error) {
	// For local MinIO or custom S3-compatible endpoints
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               cfg.Endpoint,
			HostnameImmutable: true,
		}, nil
	})

	awscfg, err := awscfg.LoadDefaultConfig(context.Background(),
		awscfg.WithRegion(cfg.Region),
		awscfg.WithEndpointResolverWithOptions(customResolver),
		awscfg.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(awscfg)

	// Ensure bucket exists
	exists, err := client.HeadBucket(context.Background(), &s3.HeadBucketInput{
		Bucket: aws.String(cfg.Bucket),
	})
	if err != nil || exists == nil {
		// Create bucket if it doesn't exist
		_, err = client.CreateBucket(context.Background(), &s3.CreateBucketInput{
			Bucket: aws.String(cfg.Bucket),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create bucket %s: %w", cfg.Bucket, err)
		}
	}

	return &S3Storage{
		client: client,
		bucket: cfg.Bucket,
	}, nil
}

// Upload stores a file and returns its URL
func (s *S3Storage) Upload(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file %s: %w", key, err)
	}

	return fmt.Sprintf("s3://%s/%s", s.bucket, key), nil
}

// Download retrieves a file's contents by its key
func (s *S3Storage) Download(ctx context.Context, key string) ([]byte, error) {
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download file %s: %w", key, err)
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read file body %s: %w", key, err)
	}

	return data, nil
}

// Delete removes a file by its key
func (s *S3Storage) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete file %s: %w", key, err)
	}

	return nil
}

// GetURL generates a presigned URL for a file
func (s *S3Storage) GetURL(ctx context.Context, key string, expiresIn int) (string, error) {
	presignClient := s3.NewPresignClient(s.client)

	req, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = time.Duration(expiresIn) * time.Second
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL for %s: %w", key, err)
	}

	return req.URL, nil
}
