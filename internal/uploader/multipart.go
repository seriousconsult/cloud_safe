package uploader

import (
	"context"
	"fmt"
	"io"
	"sync"

	"cloudarchiver/internal/logger"
	"cloudarchiver/internal/progress"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// MultipartUpload represents an ongoing multipart upload
type MultipartUpload struct {
	client     *s3.Client
	bucket     string
	key        string
	uploadID   string
	parts      []types.CompletedPart
	partNumber int32
	mu         sync.Mutex
	logger     *logger.Logger
	tracker    *progress.Tracker
}

// NewMultipartUpload creates a new multipart upload
func NewMultipartUpload(client *s3.Client, bucket, key string, logger *logger.Logger, tracker *progress.Tracker) (*MultipartUpload, error) {
	input := &s3.CreateMultipartUploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	output, err := client.CreateMultipartUpload(context.Background(), input)
	if err != nil {
		return nil, fmt.Errorf("failed to create multipart upload: %w", err)
	}

	return &MultipartUpload{
		client:     client,
		bucket:     bucket,
		key:        key,
		uploadID:   *output.UploadId,
		parts:      make([]types.CompletedPart, 0),
		partNumber: 1,
		logger:     logger,
		tracker:    tracker,
	}, nil
}

// UploadPart uploads a single part
func (mu *MultipartUpload) UploadPart(ctx context.Context, reader io.Reader, size int64) error {
	mu.mu.Lock()
	currentPart := mu.partNumber
	mu.partNumber++
	mu.mu.Unlock()

	mu.logger.Debugf("Uploading part %d (size: %d bytes)", currentPart, size)

	input := &s3.UploadPartInput{
		Bucket:     aws.String(mu.bucket),
		Key:        aws.String(mu.key),
		PartNumber: aws.Int32(currentPart),
		UploadId:   aws.String(mu.uploadID),
		Body:       reader,
	}

	output, err := mu.client.UploadPart(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to upload part %d: %w", currentPart, err)
	}

	// Add completed part
	mu.mu.Lock()
	mu.parts = append(mu.parts, types.CompletedPart{
		ETag:       output.ETag,
		PartNumber: aws.Int32(currentPart),
	})
	mu.mu.Unlock()

	// Update progress
	if mu.tracker != nil {
		mu.tracker.Update(size)
	}

	mu.logger.Debugf("Successfully uploaded part %d", currentPart)
	return nil
}

// Complete completes the multipart upload
func (mu *MultipartUpload) Complete(ctx context.Context) error {
	mu.mu.Lock()
	defer mu.mu.Unlock()

	mu.logger.Infof("Completing multipart upload with %d parts", len(mu.parts))

	input := &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(mu.bucket),
		Key:      aws.String(mu.key),
		UploadId: aws.String(mu.uploadID),
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: mu.parts,
		},
	}

	_, err := mu.client.CompleteMultipartUpload(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to complete multipart upload: %w", err)
	}

	mu.logger.Info("Multipart upload completed successfully")
	return nil
}

// Abort aborts the multipart upload
func (mu *MultipartUpload) Abort(ctx context.Context) error {
	mu.logger.Info("Aborting multipart upload")

	input := &s3.AbortMultipartUploadInput{
		Bucket:   aws.String(mu.bucket),
		Key:      aws.String(mu.key),
		UploadId: aws.String(mu.uploadID),
	}

	_, err := mu.client.AbortMultipartUpload(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to abort multipart upload: %w", err)
	}

	return nil
}

// GetUploadID returns the upload ID
func (mu *MultipartUpload) GetUploadID() string {
	return mu.uploadID
}

// GetCompletedParts returns the list of completed parts
func (mu *MultipartUpload) GetCompletedParts() []types.CompletedPart {
	mu.mu.Lock()
	defer mu.mu.Unlock()
	
	parts := make([]types.CompletedPart, len(mu.parts))
	copy(parts, mu.parts)
	return parts
}
