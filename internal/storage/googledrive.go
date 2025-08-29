package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/seriousconsult/cloud_safe/internal/logger"
	"github.com/seriousconsult/cloud_safe/internal/progress"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// GoogleDriveProvider implements StorageProvider for Google Drive
type GoogleDriveProvider struct {
	service *drive.Service
	config  *GoogleDriveConfig
	logger  *logger.Logger
}

// NewGoogleDriveProvider creates a new Google Drive storage provider
func NewGoogleDriveProvider(cfg *GoogleDriveConfig, logger *logger.Logger) (*GoogleDriveProvider, error) {
	logger.Infof("Google Drive Configuration:")
	logger.Infof("  Credentials Path: %s", cfg.CredentialsPath)
	logger.Infof("  Token Path: %s", cfg.TokenPath)
	logger.Infof("  Folder ID: %s", cfg.FolderID)
	logger.Infof("  Filename: %s", cfg.Filename)

	// Read credentials file
	b, err := os.ReadFile(cfg.CredentialsPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read client secret file: %w", err)
	}

	// Parse credentials
	config, err := google.ConfigFromJSON(b, drive.DriveFileScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file to config: %w", err)
	}

	// Get token
	client, err := getClient(config, cfg.TokenPath, logger)
	if err != nil {
		return nil, fmt.Errorf("unable to get Drive client: %w", err)
	}

	// Create Drive service
	service, err := drive.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Drive client: %w", err)
	}

	// Test connectivity
	_, err = service.About.Get().Fields("user").Do()
	if err != nil {
		return nil, fmt.Errorf("failed to test Google Drive connectivity: %w", err)
	}

	return &GoogleDriveProvider{
		service: service,
		config:  cfg,
		logger:  logger,
	}, nil
}

// getClient retrieves a token, saves the token, then returns the generated client
func getClient(config *oauth2.Config, tokFile string, logger *logger.Logger) (*http.Client, error) {
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok, err = getTokenFromWeb(config, logger)
		if err != nil {
			return nil, err
		}
		saveToken(tokFile, tok, logger)
	}
	return config.Client(context.Background(), tok), nil
}

// getTokenFromWeb requests a token from the web, then returns the retrieved token
func getTokenFromWeb(config *oauth2.Config, logger *logger.Logger) (*oauth2.Token, error) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	logger.Infof("Go to the following link in your browser then type the authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		return nil, fmt.Errorf("unable to read authorization code: %w", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve token from web: %w", err)
	}
	return tok, nil
}

// tokenFromFile retrieves a token from a local file
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// saveToken saves a token to a file path
func saveToken(path string, token *oauth2.Token, logger *logger.Logger) {
	logger.Infof("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		logger.Errorf("Unable to cache oauth token: %v", err)
		return
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

// GetProviderType returns the provider type
func (g *GoogleDriveProvider) GetProviderType() Provider {
	return ProviderGoogleDrive
}

// ValidateConfig validates the Google Drive configuration
func (g *GoogleDriveProvider) ValidateConfig() error {
	if g.config.CredentialsPath == "" {
		return fmt.Errorf("Google Drive credentials path is required")
	}
	if g.config.TokenPath == "" {
		return fmt.Errorf("Google Drive token path is required")
	}
	if g.config.Filename == "" {
		return fmt.Errorf("filename is required")
	}
	return nil
}

// UploadStream uploads data from a reader to Google Drive
func (g *GoogleDriveProvider) UploadStream(ctx context.Context, reader io.Reader, estimatedSize int64, tracker progress.Tracker) error {
	g.logger.Info("Starting Google Drive upload")

	// Create file metadata
	file := &drive.File{
		Name: g.config.Filename,
	}

	// Set parent folder if specified
	if g.config.FolderID != "" {
		file.Parents = []string{g.config.FolderID}
	}

	// Create progress reader wrapper
	var progressReader io.Reader = reader
	if tracker != nil {
		progressReader = &googleDriveProgressReader{
			reader:  reader,
			tracker: tracker,
		}
	}

	// Upload file
	_, err := g.service.Files.Create(file).Media(progressReader).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to upload file to Google Drive: %w", err)
	}

	g.logger.Info("Google Drive upload completed successfully")
	return nil
}

// CheckResumability checks if an upload can be resumed (Google Drive doesn't support resumable uploads in this implementation)
func (g *GoogleDriveProvider) CheckResumability(ctx context.Context) (ResumableUpload, error) {
	// Google Drive API supports resumable uploads, but for simplicity we'll return nil
	// In a full implementation, you would use the resumable upload protocol
	return nil, nil
}

// googleDriveProgressReader wraps an io.Reader to track progress
type googleDriveProgressReader struct {
	reader  io.Reader
	tracker progress.Tracker
}

func (pr *googleDriveProgressReader) Read(p []byte) (n int, err error) {
	n, err = pr.reader.Read(p)
	if n > 0 && pr.tracker != nil {
		pr.tracker.Update(int64(n))
	}
	return n, err
}
