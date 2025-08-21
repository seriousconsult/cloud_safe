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

// Compress compresses multiple sources (files or directories) to a tar stream
func (tc *TarCompressor) Compress(ctx context.Context, sourcePaths []string, writer io.Writer) error {
	tarWriter := tar.NewWriter(writer)
	defer tarWriter.Close()

	// Process each source path
	for _, sourcePath := range sourcePaths {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		info, err := os.Stat(sourcePath)
		if err != nil {
			return fmt.Errorf("failed to stat source path %s: %w", sourcePath, err)
		}

		if info.IsDir() {
			// Handle directory
			err = tc.compressDirectory(ctx, sourcePath, tarWriter)
		} else {
			// Handle single file
			err = tc.compressFile(ctx, sourcePath, info, tarWriter)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

// compressDirectory compresses a directory recursively
func (tc *TarCompressor) compressDirectory(ctx context.Context, sourcePath string, tarWriter *tar.Writer) error {
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
		
		// Use the directory name as prefix for better organization
		dirName := filepath.Base(sourcePath)
		if relPath == "." {
			header.Name = dirName + "/"
		} else {
			header.Name = dirName + "/" + strings.ReplaceAll(relPath, string(filepath.Separator), "/")
		}

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

			tc.logger.Debugf("Compressing file: %s", header.Name)

			// Stream file content with buffered copy
			if _, err := io.Copy(tarWriter, file); err != nil {
				return fmt.Errorf("failed to copy file content for %s: %w", path, err)
			}
		}

		return nil
	})
}

// compressFile compresses a single file
func (tc *TarCompressor) compressFile(ctx context.Context, filePath string, info os.FileInfo, tarWriter *tar.Writer) error {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Create tar header
	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return fmt.Errorf("failed to create tar header for %s: %w", filePath, err)
	}

	// Use just the filename for single files
	header.Name = filepath.Base(filePath)

	// Write header
	if err := tarWriter.WriteHeader(header); err != nil {
		return fmt.Errorf("failed to write tar header for %s: %w", filePath, err)
	}

	// Write file content
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	tc.logger.Debugf("Compressing file: %s", header.Name)

	// Stream file content with buffered copy
	if _, err := io.Copy(tarWriter, file); err != nil {
		return fmt.Errorf("failed to copy file content for %s: %w", filePath, err)
	}

	return nil
}

// EstimateSize estimates the total size of files to be compressed
func (tc *TarCompressor) EstimateSize(sourcePaths []string) (int64, error) {
	var totalSize int64

	for _, sourcePath := range sourcePaths {
		info, err := os.Stat(sourcePath)
		if err != nil {
			return 0, fmt.Errorf("failed to stat source path %s: %w", sourcePath, err)
		}

		if info.IsDir() {
			// Walk directory and sum file sizes
			err = filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				if info.Mode().IsRegular() {
					totalSize += info.Size()
				}

				return nil
			})
			if err != nil {
				return 0, err
			}
		} else {
			// Single file
			totalSize += info.Size()
		}
	}

	return totalSize, nil
}
