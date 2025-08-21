package pipeline

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"cloud_safe/internal/compressor"
	"cloud_safe/internal/config"
	"cloud_safe/internal/crypto"
	"cloud_safe/internal/logger"
	"cloud_safe/internal/progress"
	"cloud_safe/internal/storage"
)

// Processor orchestrates the entire pipeline
type Processor struct {
	config     *config.Config
	logger     *logger.Logger
	compressor *compressor.TarCompressor
	encryptor  *crypto.StreamEncryptor
	storage    storage.StorageProvider
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

	// Initialize storage provider
	storageProvider, err := storage.NewStorageProvider(cfg, log)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage provider: %w", err)
	}

	return &Processor{
		config:     cfg,
		logger:     log,
		compressor: comp,
		encryptor:  enc,
		storage:    storageProvider,
	}, nil
}

// Process executes the complete pipeline
func (p *Processor) Process(ctx context.Context) error {
	// Estimate total size for progress tracking
	totalSize, err := p.compressor.EstimateSize(p.config.SourcePaths)
	if err != nil {
		return fmt.Errorf("failed to estimate size: %w", err)
	}

	p.logger.Infof("Estimated size: %.2f MB", float64(totalSize)/(1024*1024))

	// Create progress tracker
	tracker := progress.NewTracker(totalSize)
	defer tracker.Finish()

	// Check for resumable uploads
	resumableUpload, err := p.storage.CheckResumability(ctx)
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
		// This defer closes the channel, unblocking the main goroutine
		// once this goroutine finishes.
		defer close(compressionDone)
		// This defer closes the writer, which will cause the reader to unblock
		// with an EOF or a pipe error.
		defer pipelineWriter.Close()

		p.logger.Debug("About to start compression in goroutine")
		err := p.compressor.Compress(ctx, p.config.SourcePaths, pipelineWriter)
		p.logger.Debugf("Compression goroutine finished with error: %v", err)
		p.logger.Debug("About to send compression result to channel")
		compressionDone <- err
		p.logger.Debug("Compression result sent to channel")
	}()

	var finalReader io.Reader = pipelineReader

	// Add encryption layer if enabled
	if p.config.Encrypt {
		encryptionReader, encryptionWriter := io.Pipe()

		// Start encryption in a goroutine
		encryptionDone := make(chan error, 1)
		go func() {
			defer close(encryptionDone) // Essential to unblock the main goroutine
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
	p.logger.Debug("Starting upload stream")
	uploadErr := p.storage.UploadStream(ctx, finalReader, totalSize, tracker)
	p.logger.Debug("Upload stream completed")

	// Close the pipeline reader to signal completion to compression goroutine
	pipelineReader.Close()

	// Wait for compression to complete
	p.logger.Debug("Waiting for compression to complete")
	p.logger.Debug("About to read from compressionDone channel")
	compressionErr := <-compressionDone
	p.logger.Debugf("Received compression result from channel: %v", compressionErr)
	if compressionErr != nil {
		return fmt.Errorf("compression failed: %w", compressionErr)
	}
	p.logger.Debug("Compression completed successfully")

	// Check upload result
	if uploadErr != nil {
		return fmt.Errorf("upload failed: %w", uploadErr)
	}

	p.logger.Debug("Process completed successfully")
	return nil
}

// ProcessWithProgress processes with detailed progress reporting
func (p *Processor) ProcessWithProgress(ctx context.Context, progressCallback func(transferred, total int64, speed float64)) error {
	// A potential bug exists here: this function currently calls itself,
	// leading to infinite recursion. It should probably call a helper function
	// or the main Process() function after setting up the progress tracker.
	totalSize, err := p.compressor.EstimateSize(p.config.SourcePaths)
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
	// Check source paths
	if len(p.config.SourcePaths) == 0 {
		return fmt.Errorf("at least one source path must be specified")
	}

	for _, sourcePath := range p.config.SourcePaths {
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			return fmt.Errorf("source path does not exist: %s", sourcePath)
		}
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
