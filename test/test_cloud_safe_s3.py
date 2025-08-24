# test_cloud_safe_s3.py
import pytest
import subprocess
import os
import shutil

# This fixture creates a temporary source directory with a file inside for testing.
# It automatically cleans up the directory after the test is done.
@pytest.fixture
def temp_test_dir(tmp_path):
    """
    Creates a temporary directory with a test file.
    'tmp_path' is a built-in pytest fixture for creating temporary paths.
    """
    source_dir = tmp_path / "test_source_dir"
    source_dir.mkdir()
    (source_dir / "test_file.txt").write_text("This is a temporary test file.")
    return str(source_dir)

def test_s3_upload_long_flags_success(temp_test_dir):
    """
    Tests a successful S3 upload using long flags.
    This assumes your Go program creates a local archive before uploading.
    """
    # The command to execute the Go program.
    command = [
        './cloud_safe', 
        '--source', temp_test_dir, 
        '--filename', 'test_archive.tar',
        '--s3-bucket', 'safe-storage-24',
    ]
    
    print(f"\nRunning command: {' '.join(command)}")
    
    # Execute the Go program and capture the output. 'text=True' ensures stdout/stderr
    # are treated as strings.
    result = subprocess.run(command, capture_output=True, text=True, check=False)

    # Assert that the command completed with no errors (exit code 0).
    assert result.returncode == 0, f"Command failed with exit code {result.returncode}.\nOutput:\n{result.stdout}\n{result.stderr}"
    
    # Assert that the success message is present in the output.
    assert "Upload completed successfully" in result.stdout
    
    # Assert that the local archive file was created.
    assert os.path.exists("test_archive.tar")

    # Clean up the created local archive file.
    os.remove("test_archive.tar")

def test_s3_upload_short_flags_success(temp_test_dir):
    """
    Tests a successful S3 upload using short flags.
    """
    command = [
        './cloud_safe', 
        '-s', temp_test_dir, 
        '-f', 'short_flags.tar',
        '-b', 'safe-storage-24',
    ]
    
    print(f"\nRunning command: {' '.join(command)}")
    
    result = subprocess.run(command, capture_output=True, text=True, check=False)
    
    assert result.returncode == 0, f"Command failed with exit code {result.returncode}.\nOutput:\n{result.stdout}\n{result.stderr}"
    assert "Upload completed successfully" in result.stdout
    assert os.path.exists("short_flags.tar")

    os.remove("short_flags.tar")

def test_missing_s3_bucket_flag(temp_test_dir):
    """
    Tests that the program fails when the --s3-bucket flag is missing.
    """
    command = [
        './cloud_safe', 
        '--source', temp_test_dir, 
        '--filename', 'missing_bucket.tar',
    ]
    
    print(f"\nRunning command: {' '.join(command)}")
    
    result = subprocess.run(command, capture_output=True, text=True, check=False)
    
    # Assert that the command failed with a non-zero exit code.
    assert result.returncode != 0, "Expected an error but the command succeeded."
    
    # Assert that a relevant error message is present in the output.
    assert "required flag(s) \"s3-bucket\" not set" in result.stderr

def test_nonexistent_s3_bucket():
    """
    Tests that the program fails when attempting to upload to a nonexistent S3 bucket.
    This test relies on the Go program to correctly handle the S3 API error.
    """
    command = [
        './cloud_safe',
        '--source', '.',  # We can use the current directory as a source.
        '--filename', 'nonexistent_bucket.tar',
        '--s3-bucket', 'this-bucket-should-not-exist-12345',
    ]

    print(f"\nRunning command: {' '.join(command)}")

    # Execute the Go program and capture the output.
    result = subprocess.run(command, capture_output=True, text=True, check=False)

    # Assert that the command failed with a non-zero exit code.
    assert result.returncode != 0, "Expected an error but the command succeeded."

    # Assert that an S3-related error message is present.
    assert "Error uploading" in result.stderr or "NoSuchBucket" in result.stderr or "AccessDenied" in result.stderr
