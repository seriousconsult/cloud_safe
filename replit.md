# Project Documentation

## Overview

CloudArchiver is a Go application for compressing, encrypting, and uploading large directories to cloud storage (AWS S3). It uses streaming processing to handle very large directories efficiently with minimal memory usage.

**Key Features:**
- Streaming TAR compression for memory efficiency
- AES-256-GCM encryption for security  
- S3 multipart uploads with resumable capability
- Concurrent worker pools for fast uploads
- Real-time progress tracking with ETA
- Buffer pooling for optimal memory management

## User Preferences

Preferred communication style: Simple, everyday language.

## System Architecture

**Core Components:**
- `cmd/root.go` - CLI interface using Cobra framework
- `internal/compressor/tar.go` - Streaming TAR compression
- `internal/crypto/stream.go` - AES-256-GCM encryption/decryption
- `internal/uploader/s3.go` - S3 multipart upload with retry logic
- `internal/pipeline/processor.go` - Orchestrates the complete pipeline
- `internal/progress/tracker.go` - Real-time progress tracking
- `internal/utils/buffers.go` - Memory-efficient buffer pooling

**Design Patterns:**
- Streaming pipeline architecture for memory efficiency
- Worker pool pattern for concurrent uploads
- Interface-based design for pluggable storage backends

## External Dependencies

**Cloud Storage:**
- AWS SDK v2 for S3 integration
- Supports AWS credentials via environment variables or profiles

**CLI & Progress:**
- Cobra for command-line interface
- Cheggaaa progress bar for visual feedback

**Build Requirements:**
- Go 1.19+
- AWS credentials (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_REGION)