package storage

import (
	"context"
	"io"

	"cloud_safe/internal/progress"
)

// Provider represents the type of storage provider
type Provider string

const (
	ProviderS3         Provider = "s3"
	ProviderGoogleDrive Provider = "googledrive"
	ProviderMega       Provider = "mega"
	ProviderMinIO      Provider = "minio"
)

// StorageProvider defines the interface that all storage providers must implement
type StorageProvider interface {
	// UploadStream uploads data from a reader to the storage provider
	UploadStream(ctx context.Context, reader io.Reader, estimatedSize int64, tracker progress.Tracker) error
	
	// CheckResumability checks if an upload can be resumed
	CheckResumability(ctx context.Context) (ResumableUpload, error)
	
	// GetProviderType returns the type of storage provider
	GetProviderType() Provider
	
	// ValidateConfig validates the provider-specific configuration
	ValidateConfig() error
}

// ResumableUpload represents a resumable upload session
type ResumableUpload interface {
	// Resume continues an interrupted upload
	Resume(ctx context.Context, reader io.Reader, tracker progress.Tracker) error
	
	// Abort cancels the upload session
	Abort(ctx context.Context) error
	
	// GetUploadedSize returns the amount of data already uploaded
	GetUploadedSize() int64
}

// Config holds configuration for all storage providers
type Config struct {
	Provider Provider `json:"provider"`
	
	// S3 Configuration
	S3Config *S3Config `json:"s3_config,omitempty"`
	
	// Google Drive Configuration
	GoogleDriveConfig *GoogleDriveConfig `json:"googledrive_config,omitempty"`
	
	// Mega Configuration
	MegaConfig *MegaConfig `json:"mega_config,omitempty"`
	
	// MinIO Configuration
	MinIOConfig *MinIOConfig `json:"minio_config,omitempty"`
}

// S3Config holds S3-specific configuration
type S3Config struct {
	Bucket     string `json:"bucket"`
	Key        string `json:"key"`
	Region     string `json:"region"`
	Profile    string `json:"profile"`
	ChunkSize  int64  `json:"chunk_size"`
	Workers    int    `json:"workers"`
	BufferSize int    `json:"buffer_size"`
	Resume     bool   `json:"resume"`
}

// GoogleDriveConfig holds Google Drive-specific configuration
type GoogleDriveConfig struct {
	CredentialsPath string `json:"credentials_path"`
	TokenPath       string `json:"token_path"`
	FolderID        string `json:"folder_id"`
	Filename        string `json:"filename"`
	ChunkSize       int64  `json:"chunk_size"`
	Resume          bool   `json:"resume"`
}

// MegaConfig holds Mega-specific configuration
type MegaConfig struct {
	Filename  string `json:"filename"`
	ChunkSize int64  `json:"chunk_size"`
	Resume    bool   `json:"resume"`
}

// MinIOConfig holds MinIO-specific configuration
type MinIOConfig struct {
	Endpoint        string `json:"endpoint"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	Bucket          string `json:"bucket"`
	Key             string `json:"key"`
	UseSSL          bool   `json:"use_ssl"`
	ChunkSize       int64  `json:"chunk_size"`
	Workers         int    `json:"workers"`
	BufferSize      int    `json:"buffer_size"`
	Resume          bool   `json:"resume"`
}
