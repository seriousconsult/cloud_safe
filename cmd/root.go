package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/seriousconsult/cloud_safe/internal/logger"
	"github.com/seriousconsult/cloud_safe/internal/pipeline"
	"github.com/seriousconsult/cloud_safe/internal/setup"

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
	Use:   "cloud_safe",
	Short: "Memory-efficient streaming compression and upload to cloud storage",
	Long: `CloudSafe is a tool for efficiently compressing, encrypting, and uploading
large directories to cloud storage services like AWS S3. It uses streaming processing
to minimize memory usage regardless of directory size.`,
	RunE: run,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Set default config file path
	defaultConfigPath := "config/config.json"
	if _, err := os.Stat(defaultConfigPath); err != nil {
		// If the default config file doesn't exist, use an empty string
		defaultConfigPath = ""
	}

	rootCmd.Flags().StringVarP(&cfgFile, "config", "c", defaultConfigPath, "Config file (default is config/config.json)")
	rootCmd.Flags().StringSliceVarP(&sourcePaths, "source", "s", []string{}, "Source files or directories to archive (can specify multiple)")
	// Leave provider empty by default so config.json can supply the default
	rootCmd.Flags().StringVarP(&storageProvider, "provider", "p", "", "Storage provider (s3, googledrive, mega, minio). If omitted, config.json default_settings.storage_provider is used; otherwise falls back to s3")
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
}

// getAWSProfile returns the AWS profile to use, defaulting to "sean"
func getAWSProfile() string {
	if profile := os.Getenv("AWS_PROFILE"); profile != "" {
		return profile
	}
	return ""
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

	// Only show debug info in verbose mode
	if verbose {
		wd, _ := os.Getwd()
		log.Debugf("Working directory: %s, Config: %s", wd, cfgFile)
	}

	// Start with a minimal config.
	cfg := &setup.Config{}
	err := cfg.LoadFromFile(cfgFile)
	if err != nil {
		log.Debugf("Error loading config file: %v", err)
		// Continue with default config if file doesn't exist
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to load config: %v", err)
		}
	}

	// Apply CLI flags only if explicitly set, so they override config.json
	if cmd.Flags().Changed("provider") {
		cfg.StorageProvider = storageProvider
	}
	if cmd.Flags().Changed("bucket") {
		cfg.S3Bucket = s3Bucket
	}
	if cmd.Flags().Changed("filename") {
		cfg.S3Filename = s3Filename
	}
	if cmd.Flags().Changed("gd-credentials") {
		cfg.GoogleDriveCredentialsPath = googleDriveCredPath
	}
	if cmd.Flags().Changed("gd-token") {
		cfg.GoogleDriveTokenPath = googleDriveTokenPath
	}
	if cmd.Flags().Changed("gd-folder") {
		cfg.GoogleDriveFolderID = googleDriveFolderID
	}
	if cmd.Flags().Changed("mega-username") {
		cfg.MegaUsername = megaUsername
	}
	if cmd.Flags().Changed("mega-password") {
		cfg.MegaPassword = megaPassword
	}
	if cmd.Flags().Changed("minio-endpoint") {
		cfg.MinIOEndpoint = minioEndpoint
	}
	if cmd.Flags().Changed("minio-access-key") {
		cfg.MinIOAccessKeyID = minioAccessKeyID
	}
	if cmd.Flags().Changed("minio-secret-key") {
		cfg.MinIOSecretAccessKey = minioSecretAccessKey
	}
	if cmd.Flags().Changed("minio-bucket") {
		cfg.MinIOBucket = minioBucket
	}
	if cmd.Flags().Changed("minio-ssl") {
		cfg.MinIOUseSSL = minioUseSSL
	}
	if cmd.Flags().Changed("workers") {
		cfg.Workers = workers
	}
	if cmd.Flags().Changed("chunk-size") {
		cfg.ChunkSize = chunkSize
	}
	if cmd.Flags().Changed("buffer-size") {
		cfg.BufferSize = bufferSize
	}
	if cmd.Flags().Changed("encrypt") {
		cfg.Encrypt = encrypt
	}
	if cmd.Flags().Changed("resume") {
		cfg.Resume = resume
	}
	
	// Always set AWS env-derived values unless config.json provided overrides
	if cfg.AWSRegion == "" {
		cfg.AWSRegion = getAWSRegion()
	}
	if cfg.AWSProfile == "" {
		cfg.AWSProfile = getAWSProfile()
	}
	// Final fallback for provider if still empty
	if cfg.StorageProvider == "" {
		cfg.StorageProvider = "s3"
		log.Debug("No storage provider specified via flags or config.json; defaulting to s3")
	}

	// Only show config in verbose mode
	if verbose {
		configJSON, _ := json.MarshalIndent(cfg, "", "  ")
		log.Debugf("Config after loading:\n%s", string(configJSON))
	}

	// Validate source paths and filename after config is loaded
	if len(sourcePaths) > 0 {
		cfg.SourcePaths = sourcePaths
	}
	if s3Filename != "" {
		cfg.S3Filename = s3Filename
	}
	
	if verbose {
		log.Debugf("Source paths: %v, S3 Filename: %s", cfg.SourcePaths, cfg.S3Filename)
	}
	
	if len(cfg.SourcePaths) == 0 {
		return fmt.Errorf("at least one source path must be specified via config file or command-line flag")
	}
	
	if cfg.S3Filename == "" {
		return fmt.Errorf("filename must be specified via config file or command-line flag")
	}

	for _, path := range cfg.SourcePaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("source path does not exist: %s", path)
		}
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

	log.Debug("About to call processor.Process()")
	if err := processor.Process(ctx); err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	log.Debug("processor.Process() completed successfully")
	log.Info("Upload completed successfully")

	// Special handling for Mega provider to prevent hanging
	if cfg.StorageProvider == "mega" {
		log.Info("Mega upload detected - forcing cleanup and exit")
		// Give a brief moment for any final cleanup
		time.Sleep(50 * time.Millisecond)
		log.Debug("About to call os.Exit(0)")
		// Force exit for Mega uploads due to library limitations
		os.Exit(0)
	}

	log.Debug("Returning from run() function")
	return nil
}
