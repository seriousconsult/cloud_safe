package storage

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

// S3MultipartUpload handles S3 multipart uploads and implements ResumableUpload
type S3MultipartUpload struct {
	client     *s3.Client
	bucket     string
	key        string
	uploadID   string
	parts      []types.CompletedPart
	partNumber int32
	logger     *logger.Logger
	tracker    progress.Tracker
	mutex      sync.Mutex
}

// NewS3MultipartUpload creates a new multipart upload
func NewS3MultipartUpload(client *s3.Client, bucket, key string, logger *logger.Logger, tracker progress.Tracker) (*S3MultipartUpload, error) {
	input := &s3.CreateMultipartUploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	output, err := client.CreateMultipartUpload(context.Background(), input)
	if err != nil {
		return nil, fmt.Errorf("failed to create multipart upload: %w", err)
	}

	logger.Infof("Created multipart upload: %s", *output.UploadId)

	return &S3MultipartUpload{
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
func (m *S3MultipartUpload) UploadPart(ctx context.Context, reader io.Reader, size int64) error {
	m.mutex.Lock()
	partNum := m.partNumber
	m.partNumber++
	m.mutex.Unlock()

	input := &s3.UploadPartInput{
		Bucket:     aws.String(m.bucket),
		Key:        aws.String(m.key),
		UploadId:   aws.String(m.uploadID),
		PartNumber: &partNum,
		Body:       reader,
	}

	output, err := m.client.UploadPart(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to upload part %d: %w", partNum, err)
	}

	m.logger.Debugf("Uploaded part %d, ETag: %s", partNum, *output.ETag)

	// Store the completed part
	m.mutex.Lock()
	m.parts = append(m.parts, types.CompletedPart{
		ETag:       output.ETag,
		PartNumber: &partNum,
	})
	m.mutex.Unlock()

	// Update progress tracker
	if m.tracker != nil {
		m.tracker.Update(size)
	}

	return nil
}

// Complete completes the multipart upload
func (m *S3MultipartUpload) Complete(ctx context.Context) error {
	m.mutex.Lock()
	parts := make([]types.CompletedPart, len(m.parts))
	copy(parts, m.parts)
	m.mutex.Unlock()

	input := &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(m.bucket),
		Key:      aws.String(m.key),
		UploadId: aws.String(m.uploadID),
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: parts,
		},
	}

	_, err := m.client.CompleteMultipartUpload(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to complete multipart upload: %w", err)
	}

	m.logger.Infof("Completed multipart upload: %s", m.uploadID)
	return nil
}

// Abort aborts the multipart upload
func (m *S3MultipartUpload) Abort(ctx context.Context) error {
	input := &s3.AbortMultipartUploadInput{
		Bucket:   aws.String(m.bucket),
		Key:      aws.String(m.key),
		UploadId: aws.String(m.uploadID),
	}

	_, err := m.client.AbortMultipartUpload(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to abort multipart upload: %w", err)
	}

	m.logger.Infof("Aborted multipart upload: %s", m.uploadID)
	return nil
}

// Resume continues an interrupted upload (implements ResumableUpload interface)
func (m *S3MultipartUpload) Resume(ctx context.Context, reader io.Reader, tracker progress.Tracker) error {
	m.tracker = tracker
	
	// Calculate already uploaded size
	var uploadedSize int64
	for range m.parts {
		// Estimate part size (this is approximate since we don't store actual sizes)
		uploadedSize += 5 * 1024 * 1024 // Assume 5MB per part (minimum S3 part size)
	}
	
	// Update tracker with already uploaded data
	if tracker != nil {
		tracker.Update(uploadedSize)
	}
	
	m.logger.Infof("Resuming upload from part %d", m.partNumber)
	
	// Continue uploading remaining parts
	// This would be called by the main upload logic
	return nil
}

// GetUploadedSize returns the amount of data already uploaded
func (m *S3MultipartUpload) GetUploadedSize() int64 {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// Estimate uploaded size based on number of parts
	// This is approximate since we don't store actual part sizes
	return int64(len(m.parts)) * 5 * 1024 * 1024 // Assume 5MB per part
}
