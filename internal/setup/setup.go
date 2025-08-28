package config

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

// FileConfig represents the JSON configuration file structure
type FileConfig struct {
	StorageProviders map[string]interface{} `json:"storage_providers"`
	DefaultSettings struct {
		StorageProvider string `json:"storage_provider"`
		Workers 	int 	`json:"workers"`
		ChunkSize 	int64 	`json:"chunk_size"`
		BufferSize 	int 	`json:"buffer_size"`
		Encrypt 	bool 	`json:"encrypt"`
		Resume 		bool 	`json:"resume"`
		EncryptionKey 	string `json:"encryption_key"`
		// New fields added
		SourcePath string `json:"source_path"`
		S3Filename string `json:"s3_filename"`
	} `json:"default_settings"`
}

// LoadFromFile loads configuration from a JSON file and merges with existing config
func (c *Config) LoadFromFile(configPath string) error {
	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Config file doesn't exist, return without error (use defaults/CLI flags)
		return nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	var fileConfig FileConfig
	if err := json.Unmarshal(data, &fileConfig); err != nil {
		return err
	}

	// Apply default settings if not already set
	if c.StorageProvider == "" && fileConfig.DefaultSettings.StorageProvider != "" {
		c.StorageProvider = fileConfig.DefaultSettings.StorageProvider
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
	// New logic to handle source_path and s3_filename
	if len(c.SourcePaths) == 0 && fileConfig.DefaultSettings.SourcePath != "" {
		// Split the single string from the config file into a slice of strings
		c.SourcePaths = strings.Split(fileConfig.DefaultSettings.SourcePath, ",")
	}
	if c.S3Filename == "" && fileConfig.DefaultSettings.S3Filename != "" {
		c.S3Filename = fileConfig.DefaultSettings.S3Filename
	}

	// Apply provider-specific settings
	if providerConfig, exists := fileConfig.StorageProviders[c.StorageProvider]; exists {
		switch c.StorageProvider {
		case "s3":
			if s3Config, ok := providerConfig.(map[string]interface{}); ok {
				if c.S3Bucket == "" {
					if bucket, ok := s3Config["bucket"].(string); ok {
						c.S3Bucket = bucket
					}
				}
				if c.AWSRegion == "" {
					if region, ok := s3Config["region"].(string); ok {
						c.AWSRegion = region
					}
				}
				if c.AWSProfile == "" {
					if profile, ok := s3Config["profile"].(string); ok {
						c.AWSProfile = profile
					}
				}
				// The new S3Filename is a default setting, so it's handled above
			}
		case "googledrive":
			if gdConfig, ok := providerConfig.(map[string]interface{}); ok {
				if c.GoogleDriveCredentialsPath == "" {
					if credPath, ok := gdConfig["credentials_path"].(string); ok {
						c.GoogleDriveCredentialsPath = credPath
					}
				}
				if c.GoogleDriveTokenPath == "" {
					if tokenPath, ok := gdConfig["token_path"].(string); ok {
						c.GoogleDriveTokenPath = tokenPath
					}
				}
				if c.GoogleDriveFolderID == "" {
					if folderID, ok := gdConfig["folder_id"].(string); ok {
						c.GoogleDriveFolderID = folderID
					}
				}
			}
		case "mega":
			if megaConfig, ok := providerConfig.(map[string]interface{}); ok {
				if c.MegaUsername == "" {
					if username, ok := megaConfig["username"].(string); ok {
						c.MegaUsername = username
					}
				}
				if c.MegaPassword == "" {
					if password, ok := megaConfig["password"].(string); ok {
						c.MegaPassword = password
					}
				}
			}
		case "minio":
			if minioConfig, ok := providerConfig.(map[string]interface{}); ok {
				if c.MinIOEndpoint == "" {
					if endpoint, ok := minioConfig["endpoint"].(string); ok {
						c.MinIOEndpoint = endpoint
					}
				}
				if c.MinIOAccessKeyID == "" {
					if accessKey, ok := minioConfig["access_key_id"].(string); ok {
						c.MinIOAccessKeyID = accessKey
					}
				}
				if c.MinIOSecretAccessKey == "" {
					if secretKey, ok := minioConfig["secret_access_key"].(string); ok {
						c.MinIOSecretAccessKey = secretKey
					}
				}
				if c.MinIOBucket == "" {
					if bucket, ok := minioConfig["bucket"].(string); ok {
						c.MinIOBucket = bucket
					}
				}
			}
		}
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