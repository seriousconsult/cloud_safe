package compressor

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"cloudarchiver/internal/logger"
)

// TarCompressor handles streaming compression of directories
type TarCompressor struct {
	logger *logger.Logger
}

// NewTarCompressor creates a new tar compressor
func NewTarCompressor(log *logger.Logger) *TarCompressor {
	return &TarCompressor{
		logger: log,
	}
}

// Compress compresses a directory to a tar stream
func (tc *TarCompressor) Compress(ctx context.Context, sourcePath string, writer io.Writer) error {
	tarWriter := tar.NewWriter(writer)
	defer tarWriter.Close()

	// Walk the directory tree
	return filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			tc.logger.Errorf("Error walking path %s: %v", path, err)
			return err
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("failed to create tar header for %s: %w", path, err)
		}

		// Set the name to be relative to source path
		relPath, err := filepath.Rel(sourcePath, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %w", path, err)
		}
		
		// Normalize path separators for tar format
		header.Name = strings.ReplaceAll(relPath, string(filepath.Separator), "/")

		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write tar header for %s: %w", path, err)
		}

		// Write file content if it's a regular file
		if info.Mode().IsRegular() {
			file, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open file %s: %w", path, err)
			}
			defer file.Close()

			tc.logger.Debugf("Compressing file: %s", relPath)

			// Stream file content with buffered copy
			if _, err := io.Copy(tarWriter, file); err != nil {
				return fmt.Errorf("failed to copy file content for %s: %w", path, err)
			}
		}

		return nil
	})
}

// EstimateSize estimates the total size of files to be compressed
func (tc *TarCompressor) EstimateSize(sourcePath string) (int64, error) {
	var totalSize int64

	err := filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.Mode().IsRegular() {
			totalSize += info.Size()
		}

		return nil
	})

	return totalSize, err
}
