package config

import "os"

// Config holds all configuration for the application
type Config struct {
        // Source configuration
        SourcePaths []string

        // Storage provider configuration
        StorageProvider string

        // S3 configuration
        S3Bucket   string
        S3Filename string
        AWSRegion  string
        AWSProfile string

        // Google Drive configuration
        GoogleDriveCredentialsPath string
        GoogleDriveTokenPath       string
        GoogleDriveFolderID        string

        // Mega configuration
        MegaUsername string
        MegaPassword string

        // MinIO configuration
        MinIOEndpoint        string
        MinIOAccessKeyID     string
        MinIOSecretAccessKey string
        MinIOBucket          string
        MinIOUseSSL          bool

        // Processing configuration
        Workers    int
        ChunkSize  int64
        BufferSize int
        Encrypt    bool
        Resume     bool

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
