package storage

import (
	"fmt"

	"github.com/seriousconsult/cloud_safe/internal/logger"
	"github.com/seriousconsult/cloud_safe/internal/setup"
)

// NewStorageProvider creates a new storage provider based on the configuration
func NewStorageProvider(cfg *setup.Config, log *logger.Logger) (StorageProvider, error) {
	// Get the provider-specific config from the main config
	switch cfg.StorageProvider {
	case string(ProviderS3):
		// S3 provider
		s3Cfg := &S3Config{
			Bucket:     cfg.S3Bucket,
			Key:        cfg.S3Filename,
			Region:     cfg.AWSRegion,
			Profile:    cfg.AWSProfile,
			ChunkSize:  cfg.ChunkSize,
			Workers:    cfg.Workers,
			BufferSize: cfg.BufferSize,
			Resume:     cfg.Resume,
		}
		return NewS3Provider(s3Cfg, log)

	case string(ProviderGoogleDrive):
		// Google Drive provider
		gdCfg := &GoogleDriveConfig{
			CredentialsPath: cfg.GoogleDriveCredentialsPath,
			TokenPath:       cfg.GoogleDriveTokenPath,
			FolderID:        cfg.GoogleDriveFolderID,
			Filename:        cfg.S3Filename,
			ChunkSize:       cfg.ChunkSize,
			Resume:          cfg.Resume,
		}
		return NewGoogleDriveProvider(gdCfg, log)

	case string(ProviderMega):
		// Mega.nz provider
		megaCfg := &MegaConfig{
			Username:  cfg.MegaUsername,
			Password:  cfg.MegaPassword,
			Filename:  cfg.S3Filename,
			ChunkSize: cfg.ChunkSize,
			Resume:    cfg.Resume,
		}
		return NewMegaProvider(megaCfg, log)

	case string(ProviderMinIO):
		// MinIO provider
		minioCfg := &MinIOConfig{
			Endpoint:        cfg.MinIOEndpoint,
			AccessKeyID:     cfg.MinIOAccessKeyID,
			SecretAccessKey: cfg.MinIOSecretAccessKey,
			Bucket:          cfg.MinIOBucket,
			Key:             cfg.S3Filename,
			UseSSL:          cfg.MinIOUseSSL,
			ChunkSize:       cfg.ChunkSize,
			Workers:         cfg.Workers,
			BufferSize:      cfg.BufferSize,
			Resume:          cfg.Resume,
		}
		return NewMinIOProvider(minioCfg, log)

	default:
		return nil, fmt.Errorf("unsupported storage provider: %s", cfg.StorageProvider)
	}
}

// ValidateProviderConfig checks if the specified provider is properly configured
func ValidateProviderConfig(cfg *setup.Config) error {
	switch cfg.StorageProvider {
	case string(ProviderS3):
		if cfg.S3Bucket == "" {
			return fmt.Errorf("S3 bucket name is required")
		}
		if cfg.AWSRegion == "" {
			return fmt.Errorf("AWS region is required for S3")
		}

	case string(ProviderGoogleDrive):
		if cfg.GoogleDriveCredentialsPath == "" {
			return fmt.Errorf("Google Drive credentials file path is required")
		}

	case string(ProviderMega):
		if cfg.MegaUsername == "" || cfg.MegaPassword == "" {
			return fmt.Errorf("Mega.nz username and password are required")
		}

	case string(ProviderMinIO):
		if cfg.MinIOEndpoint == "" {
			return fmt.Errorf("MinIO endpoint is required")
		}
		if cfg.MinIOAccessKeyID == "" || cfg.MinIOSecretAccessKey == "" {
			return fmt.Errorf("MinIO access key and secret key are required")
		}
		if cfg.MinIOBucket == "" {
			return fmt.Errorf("MinIO bucket name is required")
		}

	default:
		return fmt.Errorf("unsupported storage provider: %s", cfg.StorageProvider)
	}

	return nil
}
