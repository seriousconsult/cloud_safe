# Cloud Safe Execution Guide

This document covers how to run the `cloud_safe` utility, best practices, and troubleshooting common issues.

---

## Usage Examples

Use these examples to quickly execute an upload to your desired storage provider. Ensure your configuration and authentication steps are complete (see [Configuration Guide](STORAGE_PROVIDER_CONFIGURATION.md)).

### AWS S3 Upload
```bash
# Basic upload
cloud_safe -s /path/to/files -p s3 -b my-bucket -f archive.tar.gz.enc

# With custom region and profile
AWS_REGION=us-west-2 cloud_safe -s /path/to/files -p s3 -b my-bucket \
  --aws-profile myprofile -f archive.tar.gz.enc



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


[Return to the Main Project README](../README.md)
[Go to storage provider configurtion guide](STORAGE_PROVIDER_CONFIGURATION.md)