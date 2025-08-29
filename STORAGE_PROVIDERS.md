# Storage Providers

CloudArchiver supports multiple storage providers with a unified interface, allowing you to securely back up your data to various cloud and self-hosted storage solutions.

## Supported Providers

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

## Usage Examples

### AWS S3 Upload
```bash
# Basic upload
cloudarchiver -s /path/to/files -p s3 -b my-bucket -f archive.tar.gz.enc

# With custom region and profile
AWS_REGION=us-west-2 cloudarchiver -s /path/to/files -p s3 -b my-bucket \
  --aws-profile myprofile -f archive.tar.gz.enc
```

### Google Drive Upload
```bash
# First-time setup (will open browser for OAuth)
cloudarchiver -s /path/to/files -p googledrive \
  --gd-credentials /path/to/client_secret.json \
  -f archive.tar.gz.enc

# Subsequent uploads (uses cached token)
cloudarchiver -s /path/to/files -p googledrive \
  --gd-folder "FOLDER_ID" \
  -f archive.tar.gz.enc
```

### Mega.nz Upload
```bash
# Basic upload
cloudarchiver -s /path/to/files -p mega \
  --mega-username your@email.com \
  --mega-password yourpassword \
  -f archive.tar.gz.enc

# With 2FA (if enabled)
cloudarchiver -s /path/to/files -p mega \
  --mega-username your@email.com \
  --mega-password yourpassword \
  --mega-2fa YOUR_2FA_CODE \
  -f archive.tar.gz.enc
```

### MinIO Upload
```bash
# Basic upload to self-hosted MinIO
cloudarchiver -s /path/to/files -p minio \
  --minio-endpoint minio.example.com:9000 \
  --minio-access-key YOUR_ACCESS_KEY \
  --minio-secret-key YOUR_SECRET_KEY \
  --minio-bucket my-bucket \
  --minio-ssl \
  -f archive.tar.gz.enc
```

## Configuration Options

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
- Environment variables: 
  - `AWS_ACCESS_KEY_ID`
  - `AWS_SECRET_ACCESS_KEY`
  - `AWS_SESSION_TOKEN`
  - `AWS_REGION`
  - `AWS_PROFILE`

### Google Drive Options
- `--gd-credentials`: Path to Google Drive OAuth client credentials JSON file
- `--gd-token`: Path to store/load OAuth token (default: `~/.cloudarchiver/gdrive_token.json`)
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

## Authentication Setup

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
   - Navigate to "APIs & Services" > "Library"
   - Search for and enable "Google Drive API"

3. **Create OAuth 2.0 Client ID**
   - Go to "APIs & Services" > "Credentials"
   - Click "Create Credentials" > "OAuth client ID"
   - Select "Desktop app" as application type
   - Download the JSON file as `client_secret.json`

4. **First Run**
   - Run the tool with `--gd-credentials` pointing to your JSON file
   - A browser will open for authentication
   - Grant the necessary permissions

### Mega.nz
1. **Create Account**
   - Sign up at [mega.nz](https://mega.nz/register)
   - Verify your email address

2. **Enable Two-Factor Authentication (Recommended)**
   - Go to "Security" in your Mega account
   - Enable 2FA and scan the QR code with an authenticator app

3. **App Password (Optional but Recommended)**
   - Create an app-specific password for CloudArchiver
   - Use this instead of your main password

### MinIO
1. **Get Access Credentials**
   - From your MinIO server admin
   - Or create via MinIO Console: "Access Keys" > "Create access key"

2. **Environment Variables**
   ```bash
   export MINIO_ACCESS_KEY=your_access_key
   export MINIO_SECRET_KEY=your_secret_key
   export MINIO_ENDPOINT=https://minio.example.com
   ```

3. **Using .netrc (Alternative)**
   Add to `~/.netrc`:
   ```
   machine minio.example.com
   login YOUR_ACCESS_KEY
   password YOUR_SECRET_KEY
   ```

## Features by Provider

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

## Error Handling & Recovery

CloudArchiver implements robust error handling and recovery mechanisms:

### Common Issues & Solutions

#### Network Issues
- **Symptom**: Uploads fail with timeout or connection errors
  - **Solution**: Check network connectivity, increase timeouts, use `--resume`
  - **Command**: `cloudarchiver ... --timeout 300 --retry 3 --resume`

#### Authentication Failures
- **Symptom**: "Access Denied" or "Invalid Credentials"
  - **Solution**: Verify credentials, check token expiration (Google Drive), regenerate keys if needed

#### Rate Limiting
- **Symptom**: "Too Many Requests" errors
  - **Solution**: Enable rate limiting, reduce concurrency
  - **Command**: `cloudarchiver ... --rate-limit 10 --workers 2`

### Recovery Options

1. **Resume Interrupted Transfers**
   ```bash
   # Will continue from last successful chunk
   cloudarchiver ... --resume
   ```

2. **Verify Uploads**
   ```bash
   # Compare local and remote checksums
   cloudarchiver verify -f archive.tar.gz.enc -p s3 -b my-bucket
   ```

3. **Logging & Debugging**
   ```bash
   # Enable debug logging
   CLOUDARCHIVER_DEBUG=1 cloudarchiver ... -v
   
   # Log to file
   cloudarchiver ... --log-file upload.log
   ```

### Provider-Specific Notes

#### AWS S3
- **Multipart Uploads**: Automatically cleans up failed multipart uploads after 7 days
- **Permissions**: Ensure IAM user has `s3:PutObject`, `s3:GetObject`, and `s3:ListBucket` permissions

#### Google Drive
- **Token Expiry**: Access tokens expire after 1 hour; refresh tokens remain valid
- **Quota Limits**: Default 750GB daily upload limit for GSuite Business/Enterprise

#### Mega.nz
- **Bandwidth Limits**: Free accounts have transfer quotas
- **Session Management**: Use `--mega-session` to avoid repeated logins

#### MinIO
- **Self-Signed Certs**: Use `--minio-insecure` for testing (not recommended for production)
- **Versioning**: Enable bucket versioning for additional protection

## Best Practices

1. **Backup Strategy**
   - Use incremental backups for large datasets
   - Implement retention policies
   - Test restore procedures regularly

2. **Security**
   - Use IAM roles instead of access keys when possible
   - Rotate credentials periodically
   - Enable MFA for all accounts

3. **Monitoring**
   - Set up CloudWatch/S3 notifications for failed uploads
   - Monitor storage usage and costs
   - Review access logs regularly

4. **Performance Tuning**
   - Adjust `--chunk-size` based on file size and network conditions
   - Increase `--workers` for high-latency connections
   - Use `--buffer-size` to optimize memory usage
