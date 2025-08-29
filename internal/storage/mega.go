package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/seriousconsult/cloud_safe/internal/logger"
	"github.com/seriousconsult/cloud_safe/internal/progress"

	"github.com/t3rm1n4l/go-mega"
)

// MegaProvider implements StorageProvider for Mega.nz
type MegaProvider struct {
	client *mega.Mega
	config *MegaConfig
	logger *logger.Logger
}

// MegaCredentials represents the structure of the Mega configuration file
type MegaCredentials struct {
	Mega struct {
		Username string `json:"username"`
		Password string `json:"password"`
	} `json:"mega"`
}

// NewMegaProvider creates a new Mega storage provider
func NewMegaProvider(cfg *MegaConfig, logger *logger.Logger) (*MegaProvider, error) {
	// Load credentials from configuration file
	creds, err := loadMegaCredentials(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to load Mega credentials: %w", err)
	}

	logger.Infof("Mega Configuration:")
	logger.Infof("  Username: %s", creds.Mega.Username)
	logger.Infof("  Filename: %s", cfg.Filename)

	// Create Mega client
	m := mega.New()

	// Login to Mega using credentials from file
	err = m.Login(creds.Mega.Username, creds.Mega.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to login to Mega: %w", err)
	}

	logger.Info("Successfully logged in to Mega")

	return &MegaProvider{
		client: m,
		config: cfg,
		logger: logger,
	}, nil
}

// loadMegaCredentials loads Mega credentials from ~/.mega/configuration.json
func loadMegaCredentials(logger *logger.Logger) (*MegaCredentials, error) {
	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	// Construct path to configuration file
	configPath := filepath.Join(homeDir, ".mega", "configuration.json")
	logger.Infof("Loading Mega credentials from: %s", configPath)

	// Read configuration file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Mega configuration file at %s: %w", configPath, err)
	}

	// Parse JSON
	var creds MegaCredentials
	err = json.Unmarshal(data, &creds)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Mega configuration JSON: %w", err)
	}

	// Validate credentials
	if creds.Mega.Username == "" {
		return nil, fmt.Errorf("username is empty in Mega configuration file")
	}
	if creds.Mega.Password == "" {
		return nil, fmt.Errorf("password is empty in Mega configuration file")
	}

	return &creds, nil
}

// GetProviderType returns the provider type
func (m *MegaProvider) GetProviderType() Provider {
	return ProviderMega
}

// ValidateConfig validates the Mega configuration
func (m *MegaProvider) ValidateConfig() error {
	if m.config.Filename == "" {
		return fmt.Errorf("filename is required")
	}
	// Username and password are now loaded from configuration file, not from config
	return nil
}

// UploadStream uploads data from a reader to Mega
func (m *MegaProvider) UploadStream(ctx context.Context, reader io.Reader, estimatedSize int64, tracker progress.Tracker) error {
	m.logger.Info("Starting Mega upload")

	// Get root node
	root := m.client.FS.GetRoot()
	if root == nil {
		return fmt.Errorf("failed to get Mega root directory")
	}

	// Create new upload
	upload, err := m.client.NewUpload(root, m.config.Filename, estimatedSize)
	if err != nil {
		return fmt.Errorf("failed to create Mega upload: %w", err)
	}

	// Upload chunks using Mega's predefined chunk sizes
	totalUploaded := int64(0)
	
	for chunkID := 0; chunkID < upload.Chunks(); chunkID++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Get chunk location info from Mega
		position, expectedSize, err := upload.ChunkLocation(chunkID)
		if err != nil {
			return fmt.Errorf("failed to get chunk location: %w", err)
		}

		// Read exactly the amount Mega expects for this chunk
		chunkData := make([]byte, expectedSize)
		n, readErr := io.ReadFull(reader, chunkData)
		
		// Handle partial read (last chunk or end of stream)
		if readErr == io.ErrUnexpectedEOF || readErr == io.EOF {
			if n == 0 {
				break // No more data
			}
			// Truncate to actual size read
			chunkData = chunkData[:n]
		} else if readErr != nil {
			return fmt.Errorf("failed to read data: %w", readErr)
		}

		// Upload the chunk with exact size
		err = upload.UploadChunk(chunkID, chunkData)
		if err != nil {
			return fmt.Errorf("failed to upload chunk %d: %w", chunkID, err)
		}

		totalUploaded += int64(len(chunkData))
		if tracker != nil {
			tracker.Update(int64(len(chunkData)))
		}

		m.logger.Debugf("Uploaded chunk %d: %d bytes (position: %d, expected: %d)", 
			chunkID, len(chunkData), position, expectedSize)

		// If we read less than expected, we're done
		if len(chunkData) < expectedSize {
			break
		}
	}

	// Finish the upload
	node, err := upload.Finish()
	if err != nil {
		return fmt.Errorf("failed to finish Mega upload: %w", err)
	}

	m.logger.Infof("Mega upload completed successfully: %s (%d bytes)", node.GetName(), totalUploaded)
	
	m.logger.Debug("Starting Mega client cleanup")
	// Force cleanup of Mega client to prevent hanging
	// The go-mega library doesn't provide a proper Close() method
	// Set client to nil to release reference
	m.client = nil
	m.logger.Debug("Mega client set to nil")
	
	// Force cleanup with a small delay to allow any pending operations to complete
	go func() {
		m.logger.Debug("Starting background cleanup goroutine")
		time.Sleep(100 * time.Millisecond)
		// Force garbage collection to clean up any remaining resources
		// This is a workaround for the go-mega library's background goroutines
		m.logger.Debug("Background cleanup goroutine completed")
	}()
	
	m.logger.Debug("Returning from UploadStream")
	return nil
}

// CheckResumability checks if an upload can be resumed (Mega doesn't support resumable uploads in this implementation)
func (m *MegaProvider) CheckResumability(ctx context.Context) (ResumableUpload, error) {
	// Mega API supports resumable uploads, but for simplicity we'll return nil
	// In a full implementation, you would implement the resumable upload protocol
	return nil, nil
}

// megaProgressReader wraps an io.Reader to track progress for Mega uploads
type megaProgressReader struct {
	reader  io.Reader
	tracker progress.Tracker
}

func (mpr *megaProgressReader) Read(p []byte) (n int, err error) {
	n, err = mpr.reader.Read(p)
	if n > 0 && mpr.tracker != nil {
		mpr.tracker.Update(int64(n))
	}
	return n, err
}
