# CloudSafe - Secure Cloud Backup Tool

[![Go Version](https://img.shields.io/badge/Go-1.24.6-blue)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

CloudSafe is a high-performance Go application for securely backing up large directories to multiple cloud storage providers. It uses streaming architecture to handle files of any size with minimal memory usage, making it ideal for both personal and enterprise backup solutions.

## Key Features

- **Streaming TAR Compression** - Process files of any size with constant memory usage
- **Military-Grade Encryption** - AES-256-GCM encryption for maximum security
- **Multiple Storage Backends** - Supports AWS S3, Google Drive, Mega, and MinIO
- **Resumable Uploads** - Continue interrupted uploads without starting over
- **Concurrent Processing** - Multi-threaded uploads for maximum speed
- **Progress Tracking** - Real-time progress with ETA and transfer rates
- **Unified Interface** - Consistent commands across all storage providers
## Table of Contents
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Usage](#usage)
- [Storage Providers](#storage-providers)
- [Examples](#examples)
- [Development](#development)
- [Troubleshooting](#troubleshooting)
- [License](#license)

## Installation

### Prerequisites
- Go 1.24.6 or later
- Valid credentials for your chosen cloud storage provider

### From Source
```bash
# Clone the repository
git clone https://github.com/yourusername/cloud_safe.git
cd cloud_safe

# Build the binary
go build -o cloud_safe

# Move to your PATH (optional)
sudo mv cloud_safe /usr/local/bin/
```

### Using Docker
```bash
docker build -t cloud_safe .
docker run -v $(pwd):/data cloud_safe [command] [flags]
```

## Quick Start

1. **Initialize Configuration**
   ```bash
   ./cloud_safe init
   ```
   This creates a default `config.json` in the current directory.

2. **Edit Configuration**
   ```bash
   # Edit the config file with your preferred editor
   nano config.json
   ```

3. **Backup a Directory**
   ```bash
   ./cloud_safe backup -s /path/to/backup -f my_backup.tar
   ```

## Configuration

CloudSafe supports configuration via:
1. Command-line flags (highest precedence)
2. Environment variables
3. Configuration file (`config.json` in working directory or specified via `--config`)

### Configuration File Format

```json
{
  "storage_providers": {
    "s3": {
      "bucket": "your-bucket-name",
      "region": "us-east-1",
      "profile": "default",
      "chunk_size": 104857600,
      "workers": 4,
      "buffer_size": 65536,
      "resume": true
    },
    "googledrive": {
      "credentials_path": "~/.config/cloud_safe/credentials.json",
      "token_path": "~/.config/cloud_safe/token.json",
      "folder_id": ""
    }
  },
  "default_settings": {
    "storage_provider": "s3",
    "encrypt": true,
    "resume": true,
    "workers": 4,
    "chunk_size": 104857600,
    "buffer_size": 65536
  }
}
```

## Usage

### Basic Commands
```bash
# Backup files to default storage provider
./cloud_safe backup -s /path/to/source -f backup_name

# List available backups
./cloud_safe list

# Restore files from backup
./cloud_safe restore -f backup_name -d /restore/path
```

### Common Options
```bash
# Use a specific config file
./cloud_safe --config /path/to/config.json [command]

# Enable verbose output
./cloud_safe -v [command]

# Show help for a command
./cloud_safe help [command]
```
Core Components:

    cmd/root.go - CLI interface using Cobra framework

    internal/compressor/tar.go - Streaming TAR compression with multi-source support

    internal/crypto/stream.go - AES-256-GCM encryption/decryption

    internal/storage/ - Storage provider abstraction layer

        interface.go - Unified storage provider interface

        factory.go - Provider factory pattern

        s3.go - AWS S3 provider with multipart uploads

        googledrive.go - Google Drive provider with OAuth2

        mega.go - Mega.nz provider

        minio.go - MinIO provider with multipart uploads

    internal/pipeline/processor.go - Orchestrates the complete pipeline

    internal/progress/tracker.go - Real-time progress tracking

    internal/utils/buffers.go - Memory-efficient buffer pooling

Design Patterns:

    Streaming pipeline architecture for memory efficiency

    Worker pool pattern for concurrent uploads

    Interface-based design for pluggable storage backends

Command Line Arguments
Mandatory flags:

    --source or -s - Source files/directories to archive (can specify multiple)

    --filename or -f - Target filename for the archive

Storage Provider Selection:

    --provider or -p - Storage provider: s3, googledrive, mega, or minio (default: s3)

AWS S3 Options:

    --bucket or -b - S3 bucket name (default: safe-storage-24)

Google Drive Options:

    --gd-credentials - Path to Google Drive credentials JSON file

    --gd-token - Path to OAuth2 token file

    --gd-folder - Google Drive folder ID (optional)

Mega Options:

    --mega-username - Mega account username

    --mega-password - Mega account password

MinIO Options:

    --minio-endpoint - MinIO endpoint (e.g., localhost:9000)

    --minio-access-key - MinIO access key ID

    --minio-secret-key - MinIO secret access key

    --minio-bucket - MinIO bucket name

    --minio-ssl - Use SSL for MinIO connection (default: false)

Processing Options:

    --workers or -w - Number of concurrent workers (default: 4)

    --chunk-size - Chunk size for uploads in bytes (default: 100MB)

    --buffer-size - Buffer size for streaming operations (default: 64KB)

    --encrypt or -e - Enable encryption (default: true)

    --resume or -r - Enable resumable uploads (default: true)

    --verbose or -v - Enable verbose logging

Default Settings (config.json):

    source_path - Set a default source path for your archives.

    s3_filename - Set a default target filename for S3 archives.

External Dependencies
Cloud Storage:

    AWS SDK v2 for S3 integration

    Google Drive API for Google Drive integration

    go-mega library for Mega.nz integration

    MinIO Go SDK for MinIO integration

    OAuth2 library for Google Drive authentication

CLI & Progress:

    Cobra for command-line interface

    Progress tracking with real-time updates

Build Requirements:

    Go 1.23+

    Storage provider credentials:

        AWS S3: AWS credentials via ~/.aws/credentials file or environment variables

        Google Drive: OAuth2 credentials JSON file and token

        Mega: Username and password

        MinIO: Access key ID, secret access key, and endpoint

Ustar
