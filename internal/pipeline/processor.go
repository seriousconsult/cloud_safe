package pipeline

import (
        "context"
        "fmt"
        "io"
        "os"
        "time"

        "cloudarchiver/internal/compressor"
        "cloudarchiver/internal/config"
        "cloudarchiver/internal/crypto"
        "cloudarchiver/internal/logger"
        "cloudarchiver/internal/progress"
        "cloudarchiver/internal/uploader"
)

// Processor orchestrates the entire pipeline
type Processor struct {
        config     *config.Config
        logger     *logger.Logger
        compressor *compressor.TarCompressor
        encryptor  *crypto.StreamEncryptor
        uploader   *uploader.S3Uploader
}

// NewProcessor creates a new processor instance
func NewProcessor(cfg *config.Config, log *logger.Logger) (*Processor, error) {
        // Initialize compressor
        comp := compressor.NewTarCompressor(log)

        // Initialize encryptor if encryption is enabled
        var enc *crypto.StreamEncryptor
        if cfg.Encrypt {
                var err error
                enc, err = crypto.NewStreamEncryptor(cfg.GetEncryptionKey())
                if err != nil {
                        return nil, fmt.Errorf("failed to create encryptor: %w", err)
                }
        }

        // Initialize uploader
        up, err := uploader.NewS3Uploader(cfg, log)
        if err != nil {
                return nil, fmt.Errorf("failed to create uploader: %w", err)
        }

        return &Processor{
                config:     cfg,
                logger:     log,
                compressor: comp,
                encryptor:  enc,
                uploader:   up,
        }, nil
}

// Process executes the complete pipeline
func (p *Processor) Process(ctx context.Context) error {
        // Estimate total size for progress tracking
        totalSize, err := p.compressor.EstimateSize(p.config.SourcePath)
        if err != nil {
                return fmt.Errorf("failed to estimate size: %w", err)
        }

        p.logger.Infof("Estimated size: %.2f MB", float64(totalSize)/(1024*1024))

        // Create progress tracker
        tracker := progress.NewTracker(totalSize)
        defer tracker.Finish()

        // Check for resumable uploads
        resumableUpload, err := p.uploader.CheckResumability(ctx)
        if err != nil {
                p.logger.Errorf("Failed to check resumability: %v", err)
        }

        if resumableUpload != nil {
                p.logger.Info("Resuming previous upload")
                // TODO: Implement resume logic by calculating already uploaded size
                // and adjusting the tracker accordingly
        }

        // Create the processing pipeline
        pipelineReader, pipelineWriter := io.Pipe()

        // Start compression in a goroutine
        compressionDone := make(chan error, 1)
        go func() {
                defer pipelineWriter.Close()
                compressionDone <- p.compressor.Compress(ctx, p.config.SourcePath, pipelineWriter)
        }()

        var finalReader io.Reader = pipelineReader

        // Add encryption layer if enabled
        if p.config.Encrypt {
                encryptionReader, encryptionWriter := io.Pipe()
                
                // Start encryption in a goroutine
                encryptionDone := make(chan error, 1)
                go func() {
                        defer encryptionWriter.Close()
                        encryptionDone <- p.encryptor.EncryptStream(pipelineReader, encryptionWriter)
                }()

                finalReader = encryptionReader

                // Monitor encryption completion
                go func() {
                        if err := <-encryptionDone; err != nil {
                                p.logger.Errorf("Encryption failed: %v", err)
                        }
                }()
        }

        // Start upload
        uploadErr := p.uploader.UploadStream(ctx, finalReader, totalSize, tracker)

        // Wait for compression to complete
        if compressionErr := <-compressionDone; compressionErr != nil {
                return fmt.Errorf("compression failed: %w", compressionErr)
        }

        // Check upload result
        if uploadErr != nil {
                return fmt.Errorf("upload failed: %w", uploadErr)
        }

        return nil
}

// ProcessWithProgress processes with detailed progress reporting
func (p *Processor) ProcessWithProgress(ctx context.Context, progressCallback func(transferred, total int64, speed float64)) error {
        // Similar to Process but with custom progress callback
        totalSize, err := p.compressor.EstimateSize(p.config.SourcePath)
        if err != nil {
                return fmt.Errorf("failed to estimate size: %w", err)
        }

        tracker := progress.NewTracker(totalSize)
        defer tracker.Finish()

        // Create a custom progress updater
        if progressCallback != nil {
                go func() {
                        for {
                                select {
                                case <-ctx.Done():
                                        return
                                default:
                                        transferred, total, _ := tracker.GetProgress()
                                        speed := tracker.GetSpeed()
                                        progressCallback(transferred, total, speed)
                                        time.Sleep(time.Second)
                                }
                        }
                }()
        }

        return p.Process(ctx)
}

// Validate validates the processor configuration
func (p *Processor) Validate() error {
        // Check source path
        if _, err := os.Stat(p.config.SourcePath); os.IsNotExist(err) {
                return fmt.Errorf("source path does not exist: %s", p.config.SourcePath)
        }

        // Validate configuration parameters
        if p.config.Workers <= 0 {
                return fmt.Errorf("workers must be greater than 0")
        }

        if p.config.ChunkSize <= 0 {
                return fmt.Errorf("chunk size must be greater than 0")
        }

        if p.config.BufferSize <= 0 {
                return fmt.Errorf("buffer size must be greater than 0")
        }

        if p.config.S3Bucket == "" {
                return fmt.Errorf("S3 bucket is required")
        }

        if p.config.S3Filename == "" {
                return fmt.Errorf("S3 filename is required")
        }

        return nil
}
