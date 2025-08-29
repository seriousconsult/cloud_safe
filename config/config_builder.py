import json
import os

# Create a mapping from string names to actual Python types.
TYPE_MAP = {
    "str": str,
    "int": int,
    "bool": bool
}

CONFIG_SCHEMA = {
    "storage_providers": {
        "s3": {
            "enabled": {"prompt": "Enable S3 storage provider? (yes/no)", "default": True, "type": "bool"},
            "bucket": {"prompt": "Enter the S3 bucket name", "default": "safe-storage-24", "type": "str"},
            "region": {"prompt": "Enter the S3 region", "default": "us-east-1", "type": "str"},
            "profile": {"prompt": "Enter the S3 profile", "default": "default", "type": "str"},
            "chunk_size": {"prompt": "Enter chunk size for S3", "default": 104857600, "type": "int"},
            "workers": {"prompt": "Enter number of S3 workers", "default": 4, "type": "int"},
            "buffer_size": {"prompt": "Enter buffer size for S3", "default": 65536, "type": "int"},
            "resume": {"prompt": "Enable S3 resume? (yes/no)", "default": True, "type": "bool"}
        },
        "googledrive": {
            "enabled": {"prompt": "Enable Google Drive storage provider? (yes/no)", "default": False, "type": "bool"},
            "token_path": {"prompt": "Enter Google Drive token path", "default": "~/.google/token.json", "type": "str"},
            "folder_id": {"prompt": "Enter Google Drive folder ID", "default": "", "type": "str"},
            "chunk_size": {"prompt": "Enter chunk size for Google Drive", "default": 104857600, "type": "int"},
            "resume": {"prompt": "Enable Google Drive resume? (yes/no)", "default": True, "type": "bool"}
        },
        "mega": {
            "enabled": {"prompt": "Enable Mega.nz storage provider? (yes/no)", "default": False, "type": "bool"},
            "password": {"prompt": "Enter Mega password", "default": "", "type": "str"},
            "chunk_size": {"prompt": "Enter chunk size for Mega", "default": 104857600, "type": "int"},
            "resume": {"prompt": "Enable Mega resume? (yes/no)", "default": True, "type": "bool"}
        },
        "minio": {
            "enabled": {"prompt": "Enable MinIO storage provider? (yes/no)", "default": False, "type": "bool"},
            "access_key_id": {"prompt": "Enter Minio access key ID", "default": "", "type": "str"},
            "secret_access_key": {"prompt": "Enter Minio secret access key", "default": "", "type": "str"},
            "bucket": {"prompt": "Enter Minio bucket name", "default": "", "type": "str"},
            "use_ssl": {"prompt": "Use SSL for Minio? (yes/no)", "default": True, "type": "bool"},
            "chunk_size": {"prompt": "Enter chunk size for Minio", "default": 104857600, "type": "int"},
            "workers": {"prompt": "Enter number of Minio workers", "default": 4, "type": "int"},
            "buffer_size": {"prompt": "Enter buffer size for Minio", "default": 65536, "type": "int"},
            "resume": {"prompt": "Enable Minio resume? (yes/no)", "default": True, "type": "bool"}
        }
    },
    "default_settings": {
        "storage_provider": {"prompt": "Enter default storage provider", "default": "s3", "type": "str"},
        "workers": {"prompt": "Enter default number of workers", "default": 4, "type": "int"},
        "chunk_size": {"prompt": "Enter default chunk size", "default": 104857600, "type": "int"},
        "buffer_size": {"prompt": "Enter default buffer size", "default": 65536, "type": "int"},
        "encrypt": {"prompt": "Enable default encryption? (yes/no)", "default": True, "type": "bool"},
        "resume": {"prompt": "Enable default resume? (yes/no)", "default": True, "type": "bool"},
        "encryption_key": {"prompt": "Enter default encryption key", "default": "", "type": "str"},
        "source_path": {"prompt": "Enter the source path to archive", "default": "", "type": "str"},
        "s3_filename": {"prompt": "Enter the target filename for the object", "default": "", "type": "str"}
    }
}

def get_user_input(prompt, default_value=None, type_cast=str):
    """
    Prompts the user for input and returns the value.
    Handles default values and type casting.
    
    Args:
        prompt (str): The question to display to the user.
        default_value (any, optional): The default value to use.
        type_cast (type, optional): The data type to convert the input to.
        
    Returns:
        The user's input, cast to the specified type.
    """
    
    # Cast default value to string for display purposes.
    default_str = str(default_value) if default_value is not None else None
    
    if default_str is not None:
        display_prompt = f"{prompt} [{default_str}]: "
    else:
        display_prompt = f"{prompt}: "
        
    while True:
        user_input = input(display_prompt)
        
        if not user_input and default_value is not None:
            return default_value
        
        try:
            # Use the TYPE_MAP to get the correct type conversion function.
            # Handle the boolean case separately for user-friendly input.
            if type_cast == bool:
                return user_input.lower() in ['yes', 'y', 'true', 't', '1']
            else:
                return type_cast(user_input)
        except ValueError:
            print(f"Invalid input. Please enter a valid {type_cast.__name__}.")
            

def filter_enabled_providers(config_data):
    """
    Filter out disabled providers from the configuration.
    
    Args:
        config_data (dict): The full configuration data.
        
    Returns:
        dict: Configuration with only enabled providers.
    """
    if "storage_providers" not in config_data:
        return config_data
        
    # Only include providers that are explicitly enabled
    enabled_providers = {}
    for provider, config in config_data["storage_providers"].items():
        if config.get("enabled", False):
            # Remove the enabled flag from the provider config
            provider_config = config.copy()
            provider_config.pop("enabled", None)
            enabled_providers[provider] = provider_config
            
    config_data["storage_providers"] = enabled_providers
    return config_data

def prompt_for_config(schema):
    """
    Recursively walks through the schema and prompts the user for each value.
    This function can handle any level of nesting.
    
    Args:
        schema (dict): The nested dictionary representing the config structure.
        
    Returns:
        A dictionary containing the user-built configuration.
    """
    config_dict = {}
    
    # Iterate through each key in the current level of the schema.
    for key, value_info in schema.items():
        if isinstance(value_info, dict) and "prompt" not in value_info:
            # If the value is a dictionary without a 'prompt' key,
            # it's a nested section. Recursively call this function.
            print(f"\n--- Setting up {key.replace('_', ' ').title()} ---")
            config_dict[key] = prompt_for_config(value_info)
        else:
            # Otherwise, it's a leaf node. Prompt for the value.
            prompt = value_info["prompt"]
            default_value = value_info.get("default")
            # Correctly map the string name to the actual type object.
            type_name = value_info.get("type", "str")
            type_cast = TYPE_MAP.get(type_name, str)
            
            # The get_user_input function handles the actual prompting and type conversion.
            config_dict[key] = get_user_input(prompt, default_value, type_cast)
            
    return config_dict

def write_config_file(config_data, filename="config.json"):
    """
    Writes the configuration data to a JSON file.
    
    Args:
        config_data (dict): The dictionary containing the config data.
        filename (str, optional): The name of the file to write to.
    """
    try:
        with open(filename, 'w') as f:
            json.dump(config_data, f, indent=4)
        print(f"\nSuccessfully created {filename}!")
    except IOError as e:
        print(f"Error writing to file {filename}: {e}")

if __name__ == "__main__":
    print("--- CloudArchiver Config Builder ---")
    print("This will help you configure storage providers for CloudArchiver.\n")
    
    if os.path.exists("config.json"):
        overwrite = input("A config.json file already exists. Overwrite? (yes/no): ")
        if overwrite.lower() not in ['yes', 'y']:
            print("Operation canceled.")
            exit()
    
    # Get the full configuration with enabled/disabled status
    full_config = prompt_for_config(CONFIG_SCHEMA)
    
    # Filter out disabled providers
    filtered_config = filter_enabled_providers(full_config)
    
    # Write the filtered configuration
    write_config_file(filtered_config)
    
    # Show summary of enabled providers
    enabled_providers = [p for p, cfg in full_config.get("storage_providers", {}).items() 
                        if cfg.get("enabled", False)]
    
    print("\n--- Configuration Summary ---")
    if enabled_providers:
        print(f"Enabled storage providers: {', '.join(enabled_providers)}")
        
        # Check if default provider is enabled
        default_provider = filtered_config.get("default_settings", {}).get("storage_provider", "")
        if default_provider and default_provider not in enabled_providers:
            print(f"\nWarning: Default provider '{default_provider}' is not enabled!")
            print("Please update the default provider in the configuration.")
    else:
        print("Warning: No storage providers are enabled!")
        print("You need to enable at least one provider to use CloudArchiver.")
    
    print("\nConfiguration complete. You can now run CloudArchiver with your settings.")
