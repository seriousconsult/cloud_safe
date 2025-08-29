package setup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings" 
	"fmt"
)

// Config holds all configuration for the application
type Config struct {
	// Source configuration
	SourcePaths []string

	// Storage provider configuration
	StorageProvider string

	// S3 configuration
	S3Bucket 	string
	S3Filename 	string
	AWSRegion 	string
	AWSProfile 	string

	// Google Drive configuration
	GoogleDriveCredentialsPath string
	GoogleDriveTokenPath 	   string
	GoogleDriveFolderID 	   string

	// Mega configuration
	MegaUsername string
	MegaPassword string

	// MinIO configuration
	MinIOEndpoint 	   string
	MinIOAccessKeyID   string
	MinIOSecretAccessKey string
	MinIOBucket 	   string
	MinIOUseSSL 	   bool

	// Processing configuration
	Workers 	int
	ChunkSize 	int64
	BufferSize 	int
	Encrypt 	bool
	Resume 		bool

	// Encryption configuration
	EncryptionKey []byte
}

// GetEncryptionKey returns the encryption key from environment or generates one
func (c *Config) GetEncryptionKey() []byte {
	if len(c.EncryptionKey) == 0 {
		// In production, this should come from a secure key management system
		// For now, use a key from environment or a default
		keyStr := os.Getenv("ENCRYPTION_KEY")
		if keyStr == "" {
			// Generate a default key (32 bytes for AES-256)
			c.EncryptionKey = []byte("default-32-byte-encryption-key!!")
		} else {
			c.EncryptionKey = []byte(keyStr)
		}

		// Ensure key is exactly 32 bytes for AES-256
		if len(c.EncryptionKey) < 32 {
			key := make([]byte, 32)
			copy(key, c.EncryptionKey)
			c.EncryptionKey = key
		} else if len(c.EncryptionKey) > 32 {
			c.EncryptionKey = c.EncryptionKey[:32]
		}
	}
	return c.EncryptionKey
}

// ProviderConfig represents the common configuration for all storage providers
type ProviderConfig struct {
	Enabled bool `json:"enabled"`
}

// S3ProviderConfig represents S3 provider configuration
type S3ProviderConfig struct {
	ProviderConfig `json:",inline"`
	Bucket     string `json:"bucket"`
	Region     string `json:"region"`
	Profile    string `json:"profile"`
	ChunkSize  int64  `json:"chunk_size"`
	Workers    int    `json:"workers"`
	BufferSize int    `json:"buffer_size"`
	Resume     bool   `json:"resume"`
}

// GoogleDriveProviderConfig represents Google Drive provider configuration
type GoogleDriveProviderConfig struct {
	ProviderConfig   `json:",inline"`
	CredentialsPath string `json:"credentials_path"`
	TokenPath       string `json:"token_path"`
	FolderID        string `json:"folder_id"`
	ChunkSize       int64  `json:"chunk_size"`
	Resume          bool   `json:"resume"`
}

// MegaProviderConfig represents Mega.nz provider configuration
type MegaProviderConfig struct {
	ProviderConfig `json:",inline"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	ChunkSize int64  `json:"chunk_size"`
	Resume    bool   `json:"resume"`
}

// MinIOProviderConfig represents MinIO provider configuration
type MinIOProviderConfig struct {
	ProviderConfig   `json:",inline"`
	Endpoint        string `json:"endpoint"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	Bucket          string `json:"bucket"`
	UseSSL          bool   `json:"use_ssl"`
	ChunkSize       int64  `json:"chunk_size"`
	Workers         int    `json:"workers"`
	BufferSize      int    `json:"buffer_size"`
	Resume          bool   `json:"resume"`
}

// StorageProvidersConfig holds all storage provider configurations
type StorageProvidersConfig struct {
	S3         *S3ProviderConfig         `json:"s3,omitempty"`
	GoogleDrive *GoogleDriveProviderConfig `json:"googledrive,omitempty"`
	Mega        *MegaProviderConfig       `json:"mega,omitempty"`
	MinIO       *MinIOProviderConfig      `json:"minio,omitempty"`
}

// FileConfig represents the JSON configuration file structure
type FileConfig struct {
	StorageProviders StorageProvidersConfig `json:"storage_providers"`
	DefaultSettings struct {
		StorageProvider string `json:"storage_provider"`
		S3Bucket       string `json:"s3_bucket"`
		Workers 	int 	`json:"workers"`
		ChunkSize 	int64 	`json:"chunk_size"`
		BufferSize 	int 	`json:"buffer_size"`
		Encrypt 	bool 	`json:"encrypt"`
		Resume 		bool 	`json:"resume"`
		EncryptionKey  string `json:"encryption_key"`
		SourcePath     string `json:"source_path"`
		S3Filename     string `json:"s3_filename"`
	} `json:"default_settings"`
}

// LoadFromFile loads configuration from a JSON file and merges with existing config
func (c *Config) LoadFromFile(configPath string) error {
	// Check if config file exists
	if configPath == "" {
		return fmt.Errorf("config file path is empty")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Config file doesn't exist, return without error (use defaults/CLI flags)
		return fmt.Errorf("config file does not exist: %s", configPath)
	}

	// Read the config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("error reading config file: %w", err)
	}

	// Parse the config file
	var fileConfig FileConfig
	if err := json.Unmarshal(data, &fileConfig); err != nil {
		return fmt.Errorf("error parsing config file: %w", err)
	}

	// Debug: Print config file path only in verbose mode
	if os.Getenv("VERBOSE") == "true" {
		fmt.Printf("Loaded config from %s\n", configPath)
	}

	// Apply default settings if not already set by CLI flags
	if c.StorageProvider == "" && fileConfig.DefaultSettings.StorageProvider != "" {
		c.StorageProvider = fileConfig.DefaultSettings.StorageProvider
	}
	if c.S3Bucket == "" && fileConfig.DefaultSettings.S3Bucket != "" {
		c.S3Bucket = fileConfig.DefaultSettings.S3Bucket
	}
	if c.Workers == 0 && fileConfig.DefaultSettings.Workers > 0 {
		c.Workers = fileConfig.DefaultSettings.Workers
	}
	if c.ChunkSize == 0 && fileConfig.DefaultSettings.ChunkSize > 0 {
		c.ChunkSize = fileConfig.DefaultSettings.ChunkSize
	}
	if c.BufferSize == 0 && fileConfig.DefaultSettings.BufferSize > 0 {
		c.BufferSize = fileConfig.DefaultSettings.BufferSize
	}
	if len(c.EncryptionKey) == 0 && fileConfig.DefaultSettings.EncryptionKey != "" {
		c.EncryptionKey = []byte(fileConfig.DefaultSettings.EncryptionKey)
	}
	if len(c.SourcePaths) == 0 && fileConfig.DefaultSettings.SourcePath != "" {
		// Split the source path by comma to handle multiple paths
		c.SourcePaths = strings.Split(fileConfig.DefaultSettings.SourcePath, ",")
	}
	if c.S3Filename == "" && fileConfig.DefaultSettings.S3Filename != "" {
		c.S3Filename = fileConfig.DefaultSettings.S3Filename
	}
	
	// Only set these if they haven't been set by CLI flags
	if !c.Encrypt {
		c.Encrypt = fileConfig.DefaultSettings.Encrypt
	}
	if !c.Resume {
		c.Resume = fileConfig.DefaultSettings.Resume
	}

	// Apply S3 provider settings if enabled
	if s3 := fileConfig.StorageProviders.S3; s3 != nil && s3.Enabled {
		c.S3Bucket = s3.Bucket
		c.AWSRegion = s3.Region
		c.AWSProfile = s3.Profile
		if s3.ChunkSize > 0 {
			c.ChunkSize = s3.ChunkSize
		}
		if s3.Workers > 0 {
			c.Workers = s3.Workers
		}
		if s3.BufferSize > 0 {
			c.BufferSize = s3.BufferSize
		}
		c.Resume = s3.Resume
	}

	// Apply Google Drive provider settings if enabled
	if gd := fileConfig.StorageProviders.GoogleDrive; gd != nil && gd.Enabled {
		c.GoogleDriveCredentialsPath = gd.CredentialsPath
		c.GoogleDriveTokenPath = gd.TokenPath
		c.GoogleDriveFolderID = gd.FolderID
		if gd.ChunkSize > 0 {
			c.ChunkSize = gd.ChunkSize
		}
		c.Resume = gd.Resume
	}

	// Apply Mega provider settings if enabled
	if mega := fileConfig.StorageProviders.Mega; mega != nil && mega.Enabled {
		c.MegaUsername = mega.Username
		c.MegaPassword = mega.Password
		if mega.ChunkSize > 0 {
			c.ChunkSize = mega.ChunkSize
		}
		c.Resume = mega.Resume
	}

	// Apply MinIO provider settings if enabled
	if minio := fileConfig.StorageProviders.MinIO; minio != nil && minio.Enabled {
		c.MinIOEndpoint = minio.Endpoint
		c.MinIOAccessKeyID = minio.AccessKeyID
		c.MinIOSecretAccessKey = minio.SecretAccessKey
		c.MinIOBucket = minio.Bucket
		c.MinIOUseSSL = minio.UseSSL
		if minio.ChunkSize > 0 {
			c.ChunkSize = minio.ChunkSize
		}
		if minio.Workers > 0 {
			c.Workers = minio.Workers
		}
		if minio.BufferSize > 0 {
			c.BufferSize = minio.BufferSize
		}
		c.Resume = minio.Resume
	}

	return nil
}

func GetDefaultConfigPath() string {

    // Get the path and store it in a variable
    path := filepath.Join("config/config.json")
    
    // Now print the variable
    fmt.Printf("Looking for config file at: %s\n", path)
    
    // Return the path
    return path
}