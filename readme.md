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
   ./cloud_safe  -s /path/to/backup -f my_backup.tar
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
./cloud_safe -s /path/to/source -f backup_name

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
## Storage Providers

### AWS S3
```bash
./cloud_safe  -s /data -p s3 -b your-bucket -f .tgz
```

### Google Drive
```bash
# First time setup
google-drive-oauth2-cli --client_id=YOUR_CLIENT_ID --client_secret=YOUR_SECRET

# Backup to Google Drive
./cloud_safe  -s /data -p googledrive --gd-folder FOLDER_ID -f backup.tgz
```

### Mega.nz
```bash
./cloud_safe  -s /data -p mega -f backup.tgz \
  --mega-username your@email.com --mega-password yourpassword
```

### MinIO
```bash
./cloud_safe  -s /data -p minio \
  --minio-endpoint localhost:9000 \
  --minio-access-key minioadmin \
  --minio-secret-key minioadmin \
  --minio-bucket backups \
  -f backup.tgz
```

## Examples

### Encrypted Backup with Progress
```bash
./cloud_safe -s /important/data -f data_backup.tgz
```

### Backup Multiple Directories
```bash
./cloud_safe -s /home/user/documents -s /home/user/pictures -f user_data.tgz
```

### Resume Failed Upload
```bash
./cloud_safe -s /large/data -f big_backup.tgz --resume
```

## Development

### Building from Source
```bash
git clone https://github.com/yourusername/cloud_safe.git
cd cloud_safe
go build -o cloud_safe
```


### Test
-s is verbose and optional

```bash
cd /cloud_safe/
python3 -m venv venv
source venv/bin/activate
pip install pytest

pytest -s

deactivate
```

### Common Issues

**Upload Fails with "Access Denied"**
- Verify your credentials have the correct permissions
- Check if the bucket exists and is accessible
- For S3, ensure your region is correct

**Slow Upload Speeds**
- Increase the number of workers: `-w 8`
- Adjust chunk size: `--chunk-size 256MB`
- Check your network connection

**Resume Not Working**
- Ensure the `--resume` flag is set
- Check that the temporary directory is writable
- Verify the upload ID is still valid (some providers expire them)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please read our [Contributing Guidelines](CONTRIBUTING.md) for details on our code of conduct and the process for submitting pull requests.

