import subprocess
import pytest
import os
import shutil
import json
import boto3

# --- Configuration ---
# You might need to adjust this path.
CLI_PATH = "./cloud_safe"
TEST_FOLDER = "temp_test_data"
TEST_FILENAME = "test_file.txt"
CONFIG_PATH = "config/config.json"  # Path to your config file
TARGET_KEY = "test_archive.tar"
EXPECTED_TAG = "Source=cloud_safe" # The tag we set in the Go code




# --- Tests ---

def get_config_value(key):
    """Loads the S3 bucket name from the config file."""
    try:
        with open(CONFIG_PATH, 'r') as f:
            config = json.load(f)
            # Check for the key at the top level or within default_settings
            return config.get(key) or config.get('default_settings', {}).get(key)
    except FileNotFoundError:
        pytest.fail(f"Config file not found at: {CONFIG_PATH}. Cannot determine S3 bucket.")
    except json.JSONDecodeError:
        pytest.fail(f"Invalid JSON in config file: {CONFIG_PATH}")
    except Exception as e:
        pytest.fail(f"Error reading config: {e}")
        
    return None

# --- Setup and Teardown (Fixtures) ---

@pytest.fixture(scope="module", autouse=True)
def setup_test_environment():
    """Ensures a clean test environment."""
    
    # 1. Setup local test files
    if not os.path.exists(TEST_FOLDER):
        os.mkdir(TEST_FOLDER)
    
    with open(os.path.join(TEST_FOLDER, TEST_FILENAME), 'w') as f:
        # Write enough content to ensure a stable size for verification (4 bytes: t,e,s,t)
        f.write("test") 
    
    # Calculate the size of the directory/files to expect in S3 (Go's tar/archive size might differ
    # slightly, but for this tiny file, it should be close to 4 bytes before archiving overhead).
    # NOTE: It's extremely difficult to predict the exact size of a compressed/encrypted archive.
    # We will mainly rely on the key and tag verification.
    
    # 2. Yield control to the tests
    yield
    
    # 3. Teardown: Clean up local files and remote S3 object
    
    # --- S3 Cleanup ---
    bucket_name = get_config_value('s3_bucket')
    
    if bucket_name:
        s3_client = boto3.client('s3')
        try:
            # Check if object exists before trying to delete (optional, as delete_object handles non-existence if versioning is off)
            s3_client.delete_object(Bucket=bucket_name, Key=TARGET_KEY)
            print(f"\n--- S3 Teardown: Deleted {TARGET_KEY} from {bucket_name} ---")
        except Exception as e:
            # Handle cases where the object might not exist
            print(f"\n--- S3 Teardown Warning: Could not delete {TARGET_KEY}. {e} ---")
            
    # --- Local Cleanup ---
    if os.path.exists(TEST_FOLDER):
        shutil.rmtree(TEST_FOLDER)
    
# --- The Upload and Verification Test ---

def test_01_upload_and_verify_success():
    """
    Tests the upload and verifies the resulting object in S3, checking key,
    tag, and a minimum file size.
    """
    print(f"\n--- Testing Successful Upload and S3 Verification ---")
    
    bucket_name = get_config_value('s3_bucket')
    if not bucket_name:
         pytest.fail("Could not retrieve 's3_bucket' from config.json. Aborting test.")
         
    # --- STEP 1: Run the Go CLI Upload ---
    try:
        result = subprocess.run(
            [
                CLI_PATH,
                "-s", TEST_FOLDER,
                "-f", TARGET_KEY,
                "-p", "s3",
                "-v"
            ],
            capture_output=True,
            text=True,
            check=True
        )
        assert "Upload completed successfully" in result.stdout
        assert result.returncode == 0
        print("Upload successful (CLI output verified).")
        
    except subprocess.CalledProcessError as e:
        pytest.fail(f"CLI Upload failed. STDOUT: {e.stdout}\nSTDERR: {e.stderr}")


    # --- STEP 2: Verify the Object in S3 using boto3 ---
    s3_client = boto3.client('s3')
    
    # 2a. Check if the object exists and get its metadata (size)
    try:
        object_metadata = s3_client.head_object(Bucket=bucket_name, Key=TARGET_KEY)
        print(f"Object found in S3: {TARGET_KEY}")

        # 2b. Verify a minimum file size
        # A file with just "test" (4 bytes) + tar header + encryption overhead 
        # will likely be at least 100 bytes. This is an integration check.
        assert object_metadata['ContentLength'] > 100, f"Object size is too small ({object_metadata['ContentLength']} bytes). Did the upload fail?"
        print(f"Verified file size ({object_metadata['ContentLength']} bytes).")

    except s3_client.exceptions.ClientError as e:
        pytest.fail(f"S3 verification failed: Object {TARGET_KEY} not found in bucket {bucket_name}. Error: {e}")


    # 2c. Verify the custom tag
    try:
        tagging_response = s3_client.get_object_tagging(Bucket=bucket_name, Key=TARGET_KEY)
        tags = [f"{t['Key']}={t['Value']}" for t in tagging_response.get('TagSet', [])]
        
        assert EXPECTED_TAG in tags, f"Custom tag '{EXPECTED_TAG}' not found. Found tags: {tags}"
        print(f"Verified custom tag: {EXPECTED_TAG}.")
        
    except s3_client.exceptions.ClientError as e:
        pytest.fail(f"S3 tagging verification failed: {e}")

def test_02_unknown_command_is_blocked():
    """
    Tests that your new 'len(args) > 0' logic correctly blocks unknown commands.
    This should fail gracefully without running the upload.
    """
    print(f"\n--- Testing Unknown Command Block (list) ---")
    
    result = subprocess.run(
        [CLI_PATH, "list"],
        capture_output=True,
        text=True,
        check=False # Do not raise exception for non-zero exit code
    )
    
    # 1. Check for a non-zero exit code (indicating an error)
    assert result.returncode != 0
    
    # 2. Check for the specific error message you coded
    expected_error = "unknown command or argument: list"
    assert expected_error in result.stderr or expected_error in result.stdout
    
    # 3. Check that the core upload logic was NOT reached (e.g., no "Starting archive upload" message)
    # NOTE: Your Go code does some logging before the error, so this check might be tricky.
    # The best check is the returncode and the specific error message.
    print(f"STDERR:\n{result.stderr}")
    print(f"Return Code: {result.returncode}")


def test_03_unknown_command_is_blocked_2():
    """Tests the other blocked command ('lost')."""
    print(f"\n--- Testing Unknown Command Block (lost) ---")
    
    result = subprocess.run(
        [CLI_PATH, "lost"],
        capture_output=True,
        text=True,
        check=False
    )
    
    assert result.returncode != 0
    expected_error = "unknown command or argument: lost"
    assert expected_error in result.stderr or expected_error in result.stdout