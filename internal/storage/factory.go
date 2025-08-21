package storage

import (
	"fmt"

	"cloud_safe/internal/config"
	"cloud_safe/internal/logger"
)

// NewStorageProvider creates a new storage provider based on the configuration
func NewStorageProvider(cfg *config.Config, log *logger.Logger) (StorageProvider, error) {
	switch cfg.StorageProvider {
	case string(ProviderS3):
		return NewS3Provider(&S3Config{
			Bucket:     cfg.S3Bucket,
			Key:        cfg.S3Filename,
			Region:     cfg.AWSRegion,
			Profile:    cfg.AWSProfile,
			ChunkSize:  cfg.ChunkSize,
			Workers:    cfg.Workers,
			BufferSize: cfg.BufferSize,
			Resume:     cfg.Resume,
		}, log)
	case string(ProviderGoogleDrive):
		return NewGoogleDriveProvider(&GoogleDriveConfig{
			CredentialsPath: cfg.GoogleDriveCredentialsPath,
			TokenPath:       cfg.GoogleDriveTokenPath,
			FolderID:        cfg.GoogleDriveFolderID,
			Filename:        cfg.S3Filename, // Reuse filename field
			ChunkSize:       cfg.ChunkSize,
			Resume:          cfg.Resume,
		}, log)
	case string(ProviderMega):
		return NewMegaProvider(&MegaConfig{
			Filename:  cfg.S3Filename, // Reuse filename field
			ChunkSize: cfg.ChunkSize,
		}, log)
	case string(ProviderMinIO):
		return NewMinIOProvider(&MinIOConfig{
			Endpoint:        cfg.MinIOEndpoint,
			AccessKeyID:     cfg.MinIOAccessKeyID,
			SecretAccessKey: cfg.MinIOSecretAccessKey,
			Bucket:          cfg.MinIOBucket,
			Key:             cfg.S3Filename, // Reuse filename field
			UseSSL:          cfg.MinIOUseSSL,
			ChunkSize:       cfg.ChunkSize,
			Workers:         cfg.Workers,
			BufferSize:      cfg.BufferSize,
			Resume:          cfg.Resume,
		}, log)
	default:
		return nil, fmt.Errorf("unsupported storage provider: %s", cfg.StorageProvider)
	}
}
