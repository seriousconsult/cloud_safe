# Storage Providers

CloudArchiver now supports multiple storage providers with a unified interface. You can upload your archives to AWS S3, Google Drive, or Mega.nz.

## Supported Providers

### AWS S3 (Default)
- **Provider ID**: `s3`
- **Features**: Multipart uploads, resumable uploads, concurrent workers
- **Authentication**: AWS credentials (profile-based or environment variables)

### Google Drive
- **Provider ID**: `googledrive`
- **Features**: OAuth2 authentication, folder organization
- **Authentication**: Service account credentials or OAuth2 flow

### Mega.nz
- **Provider ID**: `mega`
- **Features**: Username/password authentication
- **Authentication**: Mega account credentials

## Usage Examples

### AWS S3 Upload
```bash
cloudarchiver -s /path/to/files -p s3 -b my-bucket -f archive.tar.gz.enc
```

### Google Drive Upload
```bash
cloudarchiver -s /path/to/files -p googledrive \
  --gd-credentials /path/to/credentials.json \
  --gd-token /path/to/token.json \
  --gd-folder "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms" \
  -f archive.tar.gz.enc
```

### Mega Upload
```bash
cloudarchiver -s /path/to/files -p mega \
  --mega-username your@email.com \
  --mega-password yourpassword \
  -f archive.tar.gz.enc
```

## Configuration Options

### Common Options
- `-s, --source`: Source files/directories (can specify multiple)
- `-p, --provider`: Storage provider (s3, googledrive, mega)
- `-f, --filename`: Target filename
- `-w, --workers`: Number of concurrent workers (default: 4)
- `--chunk-size`: Chunk size for uploads (default: 100MB)
- `-e, --encrypt`: Enable encryption (default: true)
- `-r, --resume`: Enable resumable uploads (default: true)

### AWS S3 Options
- `-b, --bucket`: S3 bucket name
- Environment variables: `AWS_PROFILE`, `AWS_REGION`

### Google Drive Options
- `--gd-credentials`: Path to Google Drive credentials JSON file
- `--gd-token`: Path to OAuth2 token file
- `--gd-folder`: Google Drive folder ID (optional)

### Mega Options
- `--mega-username`: Mega account username
- `--mega-password`: Mega account password

## Authentication Setup

### AWS S3
1. Configure AWS CLI: `aws configure --profile yourprofile`
2. Or set environment variables: `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`

### Google Drive
1. Create a project in Google Cloud Console
2. Enable Google Drive API
3. Create credentials (OAuth2 or Service Account)
4. Download credentials JSON file
5. Run the tool - it will guide you through OAuth2 flow on first use

### Mega
1. Create a Mega account at mega.nz
2. Use your email and password with the tool

## Features by Provider

| Feature | S3 | Google Drive | Mega |
|---------|----|--------------|----- |
| Multipart Upload | ✅ | ❌ | ❌ |
| Resumable Upload | ✅ | ❌ | ❌ |
| Concurrent Workers | ✅ | ❌ | ❌ |
| Progress Tracking | ✅ | ✅ | ✅ |
| Encryption | ✅ | ✅ | ✅ |

## Error Handling

The tool includes comprehensive error handling:
- Network connectivity issues
- Authentication failures
- Storage quota exceeded
- Invalid configurations

All providers validate their configuration before starting the upload process.
