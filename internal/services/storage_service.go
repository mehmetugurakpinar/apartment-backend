package services

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"apartment-backend/internal/config"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type StorageService struct {
	client *minio.Client
	bucket string
}

func NewStorageService(cfg config.MinIOConfig) (*StorageService, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("MinIO endpoint not configured")
	}

	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	// Ensure bucket exists
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	return &StorageService{client: client, bucket: cfg.Bucket}, nil
}

// UploadFile uploads a file to MinIO and returns the URL.
func (s *StorageService) UploadFile(ctx context.Context, reader io.Reader, size int64, contentType, folder string) (string, error) {
	ext := ".jpg"
	switch contentType {
	case "image/png":
		ext = ".png"
	case "image/gif":
		ext = ".gif"
	case "image/webp":
		ext = ".webp"
	}

	objectName := fmt.Sprintf("%s/%s%s", folder, uuid.New().String(), ext)

	_, err := s.client.PutObject(ctx, s.bucket, objectName, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	// Generate a presigned URL (7 days)
	url, err := s.client.PresignedGetObject(ctx, s.bucket, objectName, 7*24*time.Hour, nil)
	if err != nil {
		// Fallback to constructing URL manually
		scheme := "http"
		if s.client.IsOnline() {
			scheme = "https"
		}
		return fmt.Sprintf("%s://%s/%s/%s", scheme, s.client.EndpointURL().Host, s.bucket, objectName), nil
	}

	return url.String(), nil
}

// GetFileURL returns a path-style URL for a stored file.
func (s *StorageService) GetFileURL(objectName string) string {
	return filepath.Join(s.bucket, objectName)
}
