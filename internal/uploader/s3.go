package uploader

import (
        "bytes"
        "context"
        "fmt"
        "io"
        "os"
        "sync"
        "time"

        "cloudarchiver/internal/config"
        "cloudarchiver/internal/logger"
        "cloudarchiver/internal/progress"
        "cloudarchiver/internal/utils"

        "github.com/aws/aws-sdk-go-v2/aws"
        awsconfig "github.com/aws/aws-sdk-go-v2/config"
        "github.com/aws/aws-sdk-go-v2/credentials"
        "github.com/aws/aws-sdk-go-v2/service/s3"
        "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3Uploader handles streaming uploads to S3
type S3Uploader struct {
        client     *s3.Client
        config     *config.Config
        logger     *logger.Logger
        bufferPool *utils.BufferPool
}

// NewS3Uploader creates a new S3 uploader
func NewS3Uploader(cfg *config.Config, logger *logger.Logger) (*S3Uploader, error) {
        // Load AWS configuration
        var awsCfg aws.Config
        var err error

        if cfg.AWSProfile != "" {
                awsCfg, err = awsconfig.LoadDefaultConfig(context.Background(),
                        awsconfig.WithRegion(cfg.AWSRegion),
                        awsconfig.WithSharedConfigProfile(cfg.AWSProfile),
                )
        } else {
                // Try to load from environment variables or instance metadata
                awsCfg, err = awsconfig.LoadDefaultConfig(context.Background(),
                        awsconfig.WithRegion(cfg.AWSRegion),
                )
                
                // If no credentials are found, check for explicit environment variables
                if err != nil {
                        accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
                        secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
                        sessionToken := os.Getenv("AWS_SESSION_TOKEN")
                        
                        if accessKey != "" && secretKey != "" {
                                creds := credentials.NewStaticCredentialsProvider(accessKey, secretKey, sessionToken)
                                awsCfg, err = awsconfig.LoadDefaultConfig(context.Background(),
                                        awsconfig.WithRegion(cfg.AWSRegion),
                                        awsconfig.WithCredentialsProvider(creds),
                                )
                        }
                }
        }

        if err != nil {
                return nil, fmt.Errorf("failed to load AWS config: %w", err)
        }

        client := s3.NewFromConfig(awsCfg)

        // Test S3 connectivity
        _, err = client.HeadBucket(context.Background(), &s3.HeadBucketInput{
                Bucket: aws.String(cfg.S3Bucket),
        })
        if err != nil {
                return nil, fmt.Errorf("failed to access S3 bucket %s: %w", cfg.S3Bucket, err)
        }

        bufferPool := utils.NewBufferPool(cfg.BufferSize)

        return &S3Uploader{
                client:     client,
                config:     cfg,
                logger:     logger,
                bufferPool: bufferPool,
        }, nil
}

// UploadStream uploads data from a reader to S3 using multipart upload for large files
func (u *S3Uploader) UploadStream(ctx context.Context, reader io.Reader, estimatedSize int64, tracker *progress.Tracker) error {
        // Check if we should use multipart upload
        if estimatedSize > u.config.ChunkSize {
                return u.uploadMultipart(ctx, reader, estimatedSize, tracker)
        }

        // Use single part upload for smaller files
        return u.uploadSinglePart(ctx, reader, tracker)
}

// uploadSinglePart uploads a file in a single part
func (u *S3Uploader) uploadSinglePart(ctx context.Context, reader io.Reader, tracker *progress.Tracker) error {
        u.logger.Info("Using single-part upload")

        // For single part uploads, we need to buffer the entire content
        var buffer bytes.Buffer
        _, err := io.Copy(&buffer, reader)
        if err != nil {
                return fmt.Errorf("failed to buffer data for single-part upload: %w", err)
        }

        input := &s3.PutObjectInput{
                Bucket: aws.String(u.config.S3Bucket),
                Key:    aws.String(u.config.S3Key),
                Body:   bytes.NewReader(buffer.Bytes()),
        }

        _, err = u.client.PutObject(ctx, input)
        if err != nil {
                return fmt.Errorf("failed to upload object: %w", err)
        }

        if tracker != nil {
                tracker.Update(int64(buffer.Len()))
        }

        return nil
}

// uploadMultipart uploads a file using multipart upload
func (u *S3Uploader) uploadMultipart(ctx context.Context, reader io.Reader, estimatedSize int64, tracker *progress.Tracker) error {
        u.logger.Info("Using multipart upload")

        // Create multipart upload
        multipart, err := NewMultipartUpload(u.client, u.config.S3Bucket, u.config.S3Key, u.logger, tracker)
        if err != nil {
                return fmt.Errorf("failed to create multipart upload: %w", err)
        }

        // Set up error handling and cleanup
        defer func() {
                if err != nil {
                        if abortErr := multipart.Abort(context.Background()); abortErr != nil {
                                u.logger.Errorf("Failed to abort multipart upload: %v", abortErr)
                        }
                }
        }()

        // Create worker pool for concurrent uploads
        partChan := make(chan partData, u.config.Workers)
        errorChan := make(chan error, u.config.Workers)
        var wg sync.WaitGroup

        // Start worker goroutines
        for i := 0; i < u.config.Workers; i++ {
                wg.Add(1)
                go u.uploadWorker(ctx, &wg, partChan, errorChan, multipart)
        }

        // Read and send parts
        go func() {
                defer close(partChan)
                
                buffer := make([]byte, u.config.ChunkSize)
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
func (u *S3Uploader) uploadWorker(ctx context.Context, wg *sync.WaitGroup, partChan <-chan partData, errorChan chan<- error, multipart *MultipartUpload) {
        defer wg.Done()

        for part := range partChan {
                select {
                case <-ctx.Done():
                        errorChan <- ctx.Err()
                        return
                default:
                }

                // Upload the part with retry logic
                err := u.uploadPartWithRetry(ctx, multipart, part.data, part.size)
                if err != nil {
                        errorChan <- err
                        return
                }
        }
}

// uploadPartWithRetry uploads a part with retry logic
func (u *S3Uploader) uploadPartWithRetry(ctx context.Context, multipart *MultipartUpload, data []byte, size int64) error {
        const maxRetries = 3
        const baseDelay = time.Second

        for attempt := 0; attempt < maxRetries; attempt++ {
                reader := bytes.NewReader(data)
                err := multipart.UploadPart(ctx, reader, size)
                if err == nil {
                        return nil
                }

                u.logger.Errorf("Upload attempt %d failed: %v", attempt+1, err)

                if attempt < maxRetries-1 {
                        delay := baseDelay * time.Duration(1<<attempt) // Exponential backoff
                        u.logger.Debugf("Retrying in %v...", delay)
                        
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
func (u *S3Uploader) CheckResumability(ctx context.Context) (*MultipartUpload, error) {
        if !u.config.Resume {
                return nil, nil
        }

        // List ongoing multipart uploads
        input := &s3.ListMultipartUploadsInput{
                Bucket: aws.String(u.config.S3Bucket),
                Prefix: aws.String(u.config.S3Key),
        }

        output, err := u.client.ListMultipartUploads(ctx, input)
        if err != nil {
                return nil, fmt.Errorf("failed to list multipart uploads: %w", err)
        }

        // Look for an existing upload for our key
        for _, upload := range output.Uploads {
                if *upload.Key == u.config.S3Key {
                        u.logger.Infof("Found resumable upload: %s", *upload.UploadId)
                        
                        multipart := &MultipartUpload{
                                client:   u.client,
                                bucket:   u.config.S3Bucket,
                                key:      u.config.S3Key,
                                uploadID: *upload.UploadId,
                                logger:   u.logger,
                        }

                        // Get existing parts
                        parts, err := u.getExistingParts(ctx, *upload.UploadId)
                        if err != nil {
                                u.logger.Errorf("Failed to get existing parts: %v", err)
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
func (u *S3Uploader) getExistingParts(ctx context.Context, uploadID string) ([]types.CompletedPart, error) {
        input := &s3.ListPartsInput{
                Bucket:   aws.String(u.config.S3Bucket),
                Key:      aws.String(u.config.S3Key),
                UploadId: aws.String(uploadID),
        }

        var parts []types.CompletedPart
        paginator := s3.NewListPartsPaginator(u.client, input)

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
