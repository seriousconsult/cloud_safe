package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"cloudarchiver/internal/logger"
	"cloudarchiver/internal/progress"
	"cloudarchiver/internal/utils"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3Provider implements StorageProvider for AWS S3
type S3Provider struct {
	client     *s3.Client
	config     *S3Config
	logger     *logger.Logger
	bufferPool *utils.BufferPool
}

// NewS3Provider creates a new S3 storage provider
func NewS3Provider(cfg *S3Config, logger *logger.Logger) (*S3Provider, error) {
	// Log AWS configuration being used
	logger.Infof("AWS S3 Configuration:")
	logger.Infof("  Region: %s", cfg.Region)
	logger.Infof("  Profile: %s", cfg.Profile)
	logger.Infof("  S3 Bucket: %s", cfg.Bucket)
	logger.Infof("  S3 Key: %s", cfg.Key)

	// Load AWS configuration
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithSharedConfigProfile(cfg.Profile),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg)

	// Test S3 connectivity
	_, err = client.HeadBucket(context.Background(), &s3.HeadBucketInput{
		Bucket: aws.String(cfg.Bucket),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to access S3 bucket %s: %w", cfg.Bucket, err)
	}

	bufferPool := utils.NewBufferPool(cfg.BufferSize)

	return &S3Provider{
		client:     client,
		config:     cfg,
		logger:     logger,
		bufferPool: bufferPool,
	}, nil
}

// GetProviderType returns the provider type
func (s *S3Provider) GetProviderType() Provider {
	return ProviderS3
}

// ValidateConfig validates the S3 configuration
func (s *S3Provider) ValidateConfig() error {
	if s.config.Bucket == "" {
		return fmt.Errorf("S3 bucket is required")
	}
	if s.config.Key == "" {
		return fmt.Errorf("S3 key is required")
	}
	if s.config.Region == "" {
		return fmt.Errorf("AWS region is required")
	}
	return nil
}

// UploadStream uploads data from a reader to S3
func (s *S3Provider) UploadStream(ctx context.Context, reader io.Reader, estimatedSize int64, tracker progress.Tracker) error {
	// Check if we should use multipart upload
	if estimatedSize > s.config.ChunkSize {
		return s.uploadMultipart(ctx, reader, estimatedSize, tracker)
	}

	// Use single part upload for smaller files
	return s.uploadSinglePart(ctx, reader, tracker)
}

// uploadSinglePart uploads a file in a single part
func (s *S3Provider) uploadSinglePart(ctx context.Context, reader io.Reader, tracker progress.Tracker) error {
	s.logger.Info("Using single-part upload")

	// For single part uploads, we need to buffer the entire content
	var buffer bytes.Buffer
	_, err := io.Copy(&buffer, reader)
	if err != nil {
		return fmt.Errorf("failed to buffer data for single-part upload: %w", err)
	}

	input := &s3.PutObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(s.config.Key),
		Body:   bytes.NewReader(buffer.Bytes()),
	}

	_, err = s.client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to upload object: %w", err)
	}

	if tracker != nil {
		tracker.Update(int64(buffer.Len()))
	}

	return nil
}

// uploadMultipart uploads a file using multipart upload
func (s *S3Provider) uploadMultipart(ctx context.Context, reader io.Reader, estimatedSize int64, tracker progress.Tracker) error {
	s.logger.Info("Using multipart upload")

	// Create multipart upload
	multipart, err := NewS3MultipartUpload(s.client, s.config.Bucket, s.config.Key, s.logger, tracker)
	if err != nil {
		return fmt.Errorf("failed to create multipart upload: %w", err)
	}

	// Set up error handling and cleanup
	defer func() {
		if err != nil {
			if abortErr := multipart.Abort(context.Background()); abortErr != nil {
				s.logger.Errorf("Failed to abort multipart upload: %v", abortErr)
			}
		}
	}()

	// Create worker pool for concurrent uploads
	partChan := make(chan partData, s.config.Workers)
	errorChan := make(chan error, s.config.Workers)
	var wg sync.WaitGroup

	// Start worker goroutines
	for i := 0; i < s.config.Workers; i++ {
		wg.Add(1)
		go s.uploadWorker(ctx, &wg, partChan, errorChan, multipart)
	}

	// Read and send parts
	go func() {
		defer close(partChan)
		
		buffer := make([]byte, s.config.ChunkSize)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			n, readErr := io.ReadFull(reader, buffer)
			if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
				if n > 0 {
					partChan <- partData{
						data: buffer[:n],
						size: int64(n),
					}
				}
				return
			}
			if readErr != nil {
				errorChan <- fmt.Errorf("failed to read data: %w", readErr)
				return
			}

			partChan <- partData{
				data: buffer[:n],
				size: int64(n),
			}
		}
	}()

	// Wait for all uploads to complete and check for errors
	go func() {
		wg.Wait()
		close(errorChan)
	}()

	// Check for errors
	for uploadErr := range errorChan {
		if uploadErr != nil {
			return uploadErr
		}
	}

	// Complete the multipart upload
	return multipart.Complete(ctx)
}

// partData represents a part to be uploaded
type partData struct {
	data []byte
	size int64
}

// uploadWorker is a worker goroutine that uploads parts
func (s *S3Provider) uploadWorker(ctx context.Context, wg *sync.WaitGroup, partChan <-chan partData, errorChan chan<- error, multipart *S3MultipartUpload) {
	defer wg.Done()

	for part := range partChan {
		select {
		case <-ctx.Done():
			errorChan <- ctx.Err()
			return
		default:
		}

		// Upload the part with retry logic
		err := s.uploadPartWithRetry(ctx, multipart, part.data, part.size)
		if err != nil {
			errorChan <- err
			return
		}
	}
}

// uploadPartWithRetry uploads a part with retry logic
func (s *S3Provider) uploadPartWithRetry(ctx context.Context, multipart *S3MultipartUpload, data []byte, size int64) error {
	const maxRetries = 3
	const baseDelay = time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		reader := bytes.NewReader(data)
		err := multipart.UploadPart(ctx, reader, size)
		if err == nil {
			return nil
		}

		s.logger.Errorf("Upload attempt %d failed: %v", attempt+1, err)

		if attempt < maxRetries-1 {
			delay := baseDelay * time.Duration(1<<attempt) // Exponential backoff
			s.logger.Debugf("Retrying in %v...", delay)
			
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
	}

	return fmt.Errorf("failed to upload part after %d attempts", maxRetries)
}

// CheckResumability checks if an upload can be resumed
func (s *S3Provider) CheckResumability(ctx context.Context) (ResumableUpload, error) {
	if !s.config.Resume {
		return nil, nil
	}

	// List ongoing multipart uploads
	input := &s3.ListMultipartUploadsInput{
		Bucket: aws.String(s.config.Bucket),
		Prefix: aws.String(s.config.Key),
	}

	output, err := s.client.ListMultipartUploads(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list multipart uploads: %w", err)
	}

	// Look for an existing upload for our key
	for _, upload := range output.Uploads {
		if *upload.Key == s.config.Key {
			s.logger.Infof("Found resumable upload: %s", *upload.UploadId)
			
			multipart := &S3MultipartUpload{
				client:   s.client,
				bucket:   s.config.Bucket,
				key:      s.config.Key,
				uploadID: *upload.UploadId,
				logger:   s.logger,
			}

			// Get existing parts
			parts, err := s.getExistingParts(ctx, *upload.UploadId)
			if err != nil {
				s.logger.Errorf("Failed to get existing parts: %v", err)
				continue
			}

			multipart.parts = parts
			multipart.partNumber = int32(len(parts) + 1)

			return multipart, nil
		}
	}

	return nil, nil
}

// getExistingParts retrieves the list of already uploaded parts
func (s *S3Provider) getExistingParts(ctx context.Context, uploadID string) ([]types.CompletedPart, error) {
	input := &s3.ListPartsInput{
		Bucket:   aws.String(s.config.Bucket),
		Key:      aws.String(s.config.Key),
		UploadId: aws.String(uploadID),
	}

	var parts []types.CompletedPart
	paginator := s3.NewListPartsPaginator(s.client, input)

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list parts: %w", err)
		}

		for _, part := range output.Parts {
			parts = append(parts, types.CompletedPart{
				ETag:       part.ETag,
				PartNumber: part.PartNumber,
			})
		}
	}

	return parts, nil
}
