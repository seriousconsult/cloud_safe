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
        cfgFile               string
        sourcePaths           []string
        storageProvider       string
        s3Bucket              string
        s3Filename            string
        googleDriveCredPath   string
        googleDriveTokenPath  string
        googleDriveFolderID   string
        megaUsername          string
        megaPassword          string
        minioEndpoint         string
        minioAccessKeyID      string
        minioSecretAccessKey  string
        minioBucket           string
        minioUseSSL           bool
        workers               int
        chunkSize             int64
        bufferSize            int
        encrypt               bool
        resume                bool
        verbose               bool
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
        rootCmd.Flags().StringSliceVarP(&sourcePaths, "source", "s", []string{}, "Source files or directories to archive (can specify multiple)")
        rootCmd.Flags().StringVarP(&storageProvider, "provider", "p", "s3", "Storage provider (s3, googledrive, mega, minio)")
        rootCmd.Flags().StringVarP(&s3Bucket, "bucket", "b", "safe-storage-24", "S3 bucket name")
        rootCmd.Flags().StringVarP(&s3Filename, "filename", "f", "", "Target filename (required)")
        rootCmd.Flags().StringVar(&googleDriveCredPath, "gd-credentials", "", "Google Drive credentials JSON file path")
        rootCmd.Flags().StringVar(&googleDriveTokenPath, "gd-token", "", "Google Drive token file path")
        rootCmd.Flags().StringVar(&googleDriveFolderID, "gd-folder", "", "Google Drive folder ID (optional)")
        rootCmd.Flags().StringVar(&megaUsername, "mega-username", "", "Mega username")
        rootCmd.Flags().StringVar(&megaPassword, "mega-password", "", "Mega password")
        rootCmd.Flags().StringVar(&minioEndpoint, "minio-endpoint", "", "MinIO endpoint (e.g., localhost:9000)")
        rootCmd.Flags().StringVar(&minioAccessKeyID, "minio-access-key", "", "MinIO access key ID")
        rootCmd.Flags().StringVar(&minioSecretAccessKey, "minio-secret-key", "", "MinIO secret access key")
        rootCmd.Flags().StringVar(&minioBucket, "minio-bucket", "", "MinIO bucket name")
        rootCmd.Flags().BoolVar(&minioUseSSL, "minio-ssl", false, "Use SSL for MinIO connection")
        rootCmd.Flags().IntVarP(&workers, "workers", "w", 4, "Number of concurrent workers")
        rootCmd.Flags().Int64Var(&chunkSize, "chunk-size", 100*1024*1024, "Chunk size for multipart upload (bytes)")
        rootCmd.Flags().IntVar(&bufferSize, "buffer-size", 64*1024, "Buffer size for streaming operations (bytes)")
        rootCmd.Flags().BoolVarP(&encrypt, "encrypt", "e", true, "Enable encryption")
        rootCmd.Flags().BoolVarP(&resume, "resume", "r", true, "Enable resumable uploads")
        rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")

        rootCmd.MarkFlagRequired("source")
        rootCmd.MarkFlagRequired("filename")
        
}

// getAWSProfile returns the AWS profile to use, defaulting to "sean"
func getAWSProfile() string {
        if profile := os.Getenv("AWS_PROFILE"); profile != "" {
                return profile
        }
        return "sean"
}

// getAWSRegion returns the AWS region to use, defaulting to "us-east-1"
func getAWSRegion() string {
        if region := os.Getenv("AWS_REGION"); region != "" {
                return region
        }
        return "us-east-1"
}

func run(cmd *cobra.Command, args []string) error {
        // Initialize logger
        log := logger.New(verbose)

        // Validate source paths
        if len(sourcePaths) == 0 {
                return fmt.Errorf("at least one source path must be specified")
        }

        for _, path := range sourcePaths {
                if _, err := os.Stat(path); os.IsNotExist(err) {
                        return fmt.Errorf("source path does not exist: %s", path)
                }
        }

        // Load configuration
        cfg := &config.Config{
                SourcePaths:                 sourcePaths,
                StorageProvider:             storageProvider,
                S3Bucket:                    s3Bucket,
                S3Filename:                  s3Filename,
                GoogleDriveCredentialsPath:  googleDriveCredPath,
                GoogleDriveTokenPath:        googleDriveTokenPath,
                GoogleDriveFolderID:         googleDriveFolderID,
                MegaUsername:                megaUsername,
                MegaPassword:                megaPassword,
                MinIOEndpoint:               minioEndpoint,
                MinIOAccessKeyID:            minioAccessKeyID,
                MinIOSecretAccessKey:        minioSecretAccessKey,
                MinIOBucket:                 minioBucket,
                MinIOUseSSL:                 minioUseSSL,
                Workers:                     workers,
                ChunkSize:                   chunkSize,
                BufferSize:                  bufferSize,
                Encrypt:                     encrypt,
                Resume:                      resume,
                AWSRegion:                   getAWSRegion(),
                AWSProfile:                  getAWSProfile(),
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

        log.Infof("Starting archive upload: %v -> %s://%s", cfg.SourcePaths, cfg.StorageProvider, cfg.S3Filename)
        
        if err := processor.Process(ctx); err != nil {
                return fmt.Errorf("upload failed: %w", err)
        }

        log.Info("Upload completed successfully")
        return nil
}

