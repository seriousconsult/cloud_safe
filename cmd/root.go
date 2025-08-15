package cmd

import (
        "context"
        "fmt"
        "os"
        "os/signal"
        "syscall"

        "cloudarchiver/internal/config"
        "cloudarchiver/internal/logger"
        "cloudarchiver/internal/pipeline"

        "github.com/spf13/cobra"
)

var (
        cfgFile    string
        sourcePath string
        s3Bucket   string
        s3Key      string
        workers    int
        chunkSize  int64
        bufferSize int
        encrypt    bool
        resume     bool
        verbose    bool
)

var rootCmd = &cobra.Command{
        Use:   "cloudarchiver",
        Short: "Memory-efficient streaming compression and upload to cloud storage",
        Long: `CloudArchiver is a tool for efficiently compressing, encrypting, and uploading
large directories to cloud storage services like AWS S3. It uses streaming processing
to minimize memory usage regardless of directory size.`,
        RunE: run,
}


func Execute() error {
        return rootCmd.Execute()
}

func init() {
        rootCmd.Flags().StringVarP(&sourcePath, "source", "s", "", "Source directory to archive (required)")
        rootCmd.Flags().StringVarP(&s3Bucket, "bucket", "b", "", "S3 bucket name (required)")
        rootCmd.Flags().StringVarP(&s3Key, "key", "k", "", "S3 object key (required)")
        rootCmd.Flags().IntVarP(&workers, "workers", "w", 4, "Number of concurrent workers")
        rootCmd.Flags().Int64Var(&chunkSize, "chunk-size", 100*1024*1024, "Chunk size for multipart upload (bytes)")
        rootCmd.Flags().IntVar(&bufferSize, "buffer-size", 64*1024, "Buffer size for streaming operations (bytes)")
        rootCmd.Flags().BoolVarP(&encrypt, "encrypt", "e", true, "Enable encryption")
        rootCmd.Flags().BoolVarP(&resume, "resume", "r", true, "Enable resumable uploads")
        rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")

        rootCmd.MarkFlagRequired("source")
        rootCmd.MarkFlagRequired("bucket")
        rootCmd.MarkFlagRequired("key")
        
}

func run(cmd *cobra.Command, args []string) error {
        // Initialize logger
        log := logger.New(verbose)

        // Load configuration
        cfg := &config.Config{
                SourcePath: sourcePath,
                S3Bucket:   s3Bucket,
                S3Key:      s3Key,
                Workers:    workers,
                ChunkSize:  chunkSize,
                BufferSize: bufferSize,
                Encrypt:    encrypt,
                Resume:     resume,
                AWSRegion:  os.Getenv("AWS_REGION"),
                AWSProfile: os.Getenv("AWS_PROFILE"),
        }

        if cfg.AWSRegion == "" {
                cfg.AWSRegion = "us-east-1"
        }

        // Validate source path
        if _, err := os.Stat(cfg.SourcePath); os.IsNotExist(err) {
                return fmt.Errorf("source path does not exist: %s", cfg.SourcePath)
        }

        // Create context with cancellation
        ctx, cancel := context.WithCancel(context.Background())
        defer cancel()

        // Handle interrupts
        sigChan := make(chan os.Signal, 1)
        signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
        go func() {
                <-sigChan
                log.Info("Received interrupt signal, cancelling...")
                cancel()
        }()

        // Create and run processor
        processor, err := pipeline.NewProcessor(cfg, log)
        if err != nil {
                return fmt.Errorf("failed to create processor: %w", err)
        }

        log.Infof("Starting archive upload: %s -> s3://%s/%s", cfg.SourcePath, cfg.S3Bucket, cfg.S3Key)
        
        if err := processor.Process(ctx); err != nil {
                return fmt.Errorf("upload failed: %w", err)
        }

        log.Info("Upload completed successfully")
        return nil
}

