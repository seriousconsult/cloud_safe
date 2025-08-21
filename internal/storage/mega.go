package storage

import (
	"context"
	"fmt"
	"io"

	"cloudarchiver/internal/logger"
	"cloudarchiver/internal/progress"

	"github.com/t3rm1n4l/go-mega"
)

// MegaProvider implements StorageProvider for Mega.nz
type MegaProvider struct {
	client *mega.Mega
	config *MegaConfig
	logger *logger.Logger
}

// NewMegaProvider creates a new Mega storage provider
func NewMegaProvider(cfg *MegaConfig, logger *logger.Logger) (*MegaProvider, error) {
	logger.Infof("Mega Configuration:")
	logger.Infof("  Username: %s", cfg.Username)
	logger.Infof("  Filename: %s", cfg.Filename)

	// Create Mega client
	m := mega.New()

	// Login to Mega
	err := m.Login(cfg.Username, cfg.Password)
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

// GetProviderType returns the provider type
func (m *MegaProvider) GetProviderType() Provider {
	return ProviderMega
}

// ValidateConfig validates the Mega configuration
func (m *MegaProvider) ValidateConfig() error {
	if m.config.Username == "" {
		return fmt.Errorf("Mega username is required")
	}
	if m.config.Password == "" {
		return fmt.Errorf("Mega password is required")
	}
	if m.config.Filename == "" {
		return fmt.Errorf("filename is required")
	}
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

	// Create progress channel for tracking
	progressChan := make(chan int)
	
	// Start progress tracking goroutine
	if tracker != nil {
		go func() {
			for progress := range progressChan {
				tracker.Update(int64(progress))
			}
		}()
	}

	// Upload file to root directory using correct API signature
	_, err := m.client.UploadFile(m.config.Filename, root, "", &progressChan)
	if err != nil {
		close(progressChan)
		return fmt.Errorf("failed to upload file to Mega: %w", err)
	}

	close(progressChan)
	m.logger.Info("Mega upload completed successfully")
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
