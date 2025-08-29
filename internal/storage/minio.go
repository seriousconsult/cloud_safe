package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/seriousconsult/cloud_safe/internal/logger"
	"github.com/seriousconsult/cloud_safe/internal/progress"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOProvider implements the StorageProvider interface for MinIO
type MinIOProvider struct {
	client *minio.Client
	config *MinIOConfig
	logger *logger.Logger
}

// NewMinIOProvider creates a new MinIO storage provider
func NewMinIOProvider(cfg *MinIOConfig, logger *logger.Logger) (*MinIOProvider, error) {
	logger.Infof("MinIO Configuration:")
	logger.Infof("  Endpoint: %s", cfg.Endpoint)
	logger.Infof("  Bucket: %s", cfg.Bucket)
	logger.Infof("  Key: %s", cfg.Key)
	logger.Infof("  Use SSL: %t", cfg.UseSSL)

	// Initialize MinIO client
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	// Test connectivity by checking if bucket exists
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check MinIO bucket %s: %w", cfg.Bucket, err)
	}
	if !exists {
		return nil, fmt.Errorf("MinIO bucket %s does not exist", cfg.Bucket)
	}

	return &MinIOProvider{
		client: client,
		config: cfg,
		logger: logger,
	}, nil
}

// GetProviderType returns the provider type
func (m *MinIOProvider) GetProviderType() Provider {
	return ProviderMinIO
}

// ValidateConfig validates the MinIO configuration
func (m *MinIOProvider) ValidateConfig() error {
	if m.config.Endpoint == "" {
		return fmt.Errorf("MinIO endpoint is required")
	}
	if m.config.AccessKeyID == "" {
		return fmt.Errorf("MinIO access key ID is required")
	}
	if m.config.SecretAccessKey == "" {
		return fmt.Errorf("MinIO secret access key is required")
	}
	if m.config.Bucket == "" {
		return fmt.Errorf("MinIO bucket is required")
	}
	if m.config.Key == "" {
		return fmt.Errorf("MinIO key is required")
	}
	return nil
}

// UploadStream uploads data from a reader to MinIO
func (m *MinIOProvider) UploadStream(ctx context.Context, reader io.Reader, size int64, tracker progress.Tracker) error {
	m.logger.Infof("Starting MinIO upload to %s/%s (size: %d bytes)", m.config.Bucket, m.config.Key, size)

	// Create progress reader if tracker is provided
	var finalReader io.Reader = reader
	if tracker != nil {
		finalReader = &minioProgressReader{
			reader:  reader,
			tracker: tracker,
		}
	}

	// Use PutObject which automatically handles multipart for large files
	opts := minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	}

	info, err := m.client.PutObject(ctx, m.config.Bucket, m.config.Key, finalReader, size, opts)
	if err != nil {
		return fmt.Errorf("failed to upload to MinIO: %w", err)
	}

	m.logger.Infof("Successfully uploaded to MinIO: %s (ETag: %s)", info.Key, info.ETag)
	return nil
}

// CheckResumability checks if an upload can be resumed (MinIO doesn't support resumable uploads in this implementation)
func (m *MinIOProvider) CheckResumability(ctx context.Context) (ResumableUpload, error) {
	// MinIO supports multipart uploads but for simplicity we'll return nil
	// since PutObject handles multipart automatically
	return nil, nil
}

// minioProgressReader wraps an io.Reader to provide progress tracking
type minioProgressReader struct {
	reader  io.Reader
	tracker progress.Tracker
}

func (pr *minioProgressReader) Read(p []byte) (n int, err error) {
	n, err = pr.reader.Read(p)
	if n > 0 && pr.tracker != nil {
		pr.tracker.Update(int64(n))
	}
	return n, err
}

