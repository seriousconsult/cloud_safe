# Storage Provider Configuration and Setup

This document details the configuration options, required authentication steps, and feature comparisons for each supported storage provider.

---

## Supported Providers

Cloud Safe supports multiple storage providers with a unified interface, allowing you to securely back up your data to various cloud and self-hosted storage solutions.

### AWS S3 (Default)
- **Provider ID**: `s3`
- **Features**: 
  - Multipart uploads for large files
  - Resumable uploads for interrupted transfers
  - Concurrent workers for faster transfers
  - Server-side encryption support
  - S3-compatible storage support
- **Authentication**: 
  - AWS credentials file (`~/.aws/credentials`)
  - Environment variables
  - IAM roles (EC2, ECS, etc.)

### Google Drive
- **Provider ID**: `googledrive`
- **Features**:
  - OAuth2 authentication
  - Folder organization
  - Shared drive support
  - File versioning
- **Authentication**:
  - OAuth2 client credentials
  - Service account credentials
  - Token caching for multiple sessions

### Mega.nz
- **Provider ID**: `mega`
- **Features**:
  - End-to-end encryption
  - Two-factor authentication support
  - Bandwidth limits handling
  - File versioning
- **Authentication**:
  - Email/Password credentials
  - Session tokens (temporary)

### MinIO
- **Provider ID**: `minio`
- **Features**:
  - S3-compatible API
  - Multipart uploads
  - Server-side encryption
  - Self-hosted or cloud deployment
- **Authentication**:
  - Access key/Secret key
  - IAM-style policies
  - LDAP/Active Directory integration

---

## Configuration Options

These command-line flags are used to set the parameters for the upload.

### Common Options
- `-s, --source` (required): Source files/directories (can specify multiple)
- `-p, --provider`: Storage provider (`s3`, `googledrive`, `mega`, `minio`)
- `-f, --filename` (required): Target filename
- `-w, --workers`: Number of concurrent workers (default: 4)
- `--chunk-size`: Chunk size for uploads (default: 100MB, supports units: B, KB, MB, GB)
- `-e, --encrypt`: Enable encryption (default: true)
- `-r, --resume`: Enable resumable uploads (default: true)
- `-v, --verbose`: Enable verbose output
- `--temp-dir`: Directory for temporary files (default: system temp dir)

### AWS S3 Options
- `-b, --bucket` (required): S3 bucket name
- `--s3-region`: AWS region (default: us-east-1)
- `--s3-endpoint`: Custom endpoint URL (for S3-compatible storage)
- `--s3-acl`: Canned ACL (e.g., private, public-read)
- `--s3-storage-class`: Storage class (STANDARD, STANDARD_IA, etc.)

### Google Drive Options
- `--gd-credentials`: Path to Google Drive OAuth client credentials JSON file
- `--gd-token`: Path to store/load OAuth token (default: `~/.cloud_safe/gdrive_token.json`)
- `--gd-folder`: Google Drive folder ID for uploads (default: root)
- `--gd-shared-drive`: ID of shared drive (Team Drive) to use
- `--gd-impersonate`: Email of user to impersonate (domain-wide delegation)

### Mega.nz Options
- `--mega-username`: Mega account email
- `--mega-password`: Mega account password
- `--mega-2fa`: Two-factor authentication code (if enabled)
- `--mega-session`: Path to save/load session (avoids re-login)

### MinIO Options
- `--minio-endpoint` (required): MinIO server URL (host:port)
- `--minio-access-key` (required): Access key
- `--minio-secret-key` (required): Secret key
- `--minio-bucket` (required): Bucket name
- `--minio-region`: Region (default: us-east-1)
- `--minio-ssl`: Use HTTPS (default: false)
- `--minio-insecure`: Skip SSL certificate verification (not recommended)

---

## Authentication Setup

This section provides the necessary external steps to prepare your environment or accounts for use with Cloud Safe.

### AWS S3
1. **Using AWS CLI (Recommended)**
   ```bash
   aws configure --profile yourprofile
   ```
   - Enter your AWS Access Key ID, Secret Access Key, and default region
2. **Using Environment Variables**
   ```bash
   export AWS_ACCESS_KEY_ID=your_access_key
   export AWS_SECRET_ACCESS_KEY=your_secret_key
   export AWS_REGION=us-east-1
   ```
3. **IAM Roles** (for EC2/ECS)
   - Attach appropriate IAM role to your EC2 instance or ECS task

### Google Drive
1. **Create a Google Cloud Project**
   - Go to [Google Cloud Console](https://console.cloud.google.com/)
   - Create a new project or select existing one
2. **Enable APIs**
   - Search for and enable "Google Drive API"
3. **Create OAuth 2.0 Client ID**
   - Download the JSON file as `client_secret.json`
4. **First Run**
   - Run the tool with `--gd-credentials` pointing to your JSON file. A browser will open for authentication.

### Mega.nz
1. **Create Account**
   - Sign up at [mega.nz](https://mega.nz/register)
2. **Enable Two-Factor Authentication (Recommended)**
   - Enable 2FA in your Mega account security settings.
3. **App Password (Optional but Recommended)**
   - Create an app-specific password for Cloud Safe and use this instead of your main password.

### MinIO
1. **Get Access Credentials**
   - Get from your MinIO server admin or create via MinIO Console.
2. **Environment Variables**
   ```bash
   export MINIO_ACCESS_KEY=your_access_key
   export MINIO_SECRET_KEY=your_secret_key
   export MINIO_ENDPOINT=[https://minio.example.com](https://minio.example.com)
   ```

---

## Features by Provider Reference

| Feature | S3 | Google Drive | Mega | MinIO |
|---------|----|--------------|------|-------|
| **Upload Features** | | | | |
| Multipart Upload | ✅ | ❌ | ❌ | ✅ |
| Resumable Upload | ✅ | ✅ | ❌ | ✅ |
| Concurrent Uploads | ✅ | ❌ | ❌ | ✅ |
| **Security** | | | | |
| Server-side Encryption | ✅ | ✅ | ❌ | ✅ |
| Client-side Encryption | ✅ | ✅ | ✅ | ✅ |
| Two-Factor Auth | ❌ | ✅ | ✅ | ✅ |
| **Management** | | | | |
| File Versioning | ✅ | ✅ | ✅ | ✅ |
| Shared Drives | ✅ | ✅ | ❌ | ✅ |
| Custom Metadata | ✅ | ✅ | ❌ | ✅ |
| **Performance** | | | | |
| Chunked Upload | ✅ | ✅ | ✅ | ✅ |
| Parallel Uploads | ✅ | ❌ | ❌ | ✅ |
| **Access Control** | | | | |
| IAM Integration | ✅ | ❌ | ❌ | ✅ |
| ACLs | ✅ | ✅ | ❌ | ✅ |
| **Compatibility** | | | | |
| S3 API | Native | ❌ | ❌ | Compatible |
| CLI Tools | AWS CLI | gdrive | megacmd | mc |

[Return to the Main Project README](../README.md)
[Go to storeage provider execution guide](STORAGE_PROVIDER_EXECUTION.md)