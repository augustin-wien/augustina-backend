import re
import subprocess

# Define the file path and commit hash
file_path = 'app/config/config.go'
commit_hash = subprocess.check_output(['git', 'rev-parse', '--short', 'HEAD']).decode('utf-8').strip()

# Read the content of the config.go file
with open(file_path, 'r') as f:
    content = f.read()

# Find the version line using regex
match = re.search(r'var version = "(\d+\.\d+\.\d+)-([a-f0-9]+)"', content)

if match:
    # Extract the version and commit hash
    version = match.group(1)
    old_commit_hash = match.group(2)

    # Split the version into major, minor, and patch
    version_parts = version.split('.')
    major = int(version_parts[0])
    minor = int(version_parts[1])
    patch = int(version_parts[2])

    # Increment the patch version
    patch += 1

    # Create the new version
    new_version = f"{major}.{minor}.{patch}-{commit_hash}"

    # Replace the version line with the new version
    new_content = re.sub(r'var version = "(\d+\.\d+\.\d+)-[a-f0-9]+"', f'var version = "{new_version}"', content)

    # Write the updated content back to the file
    with open(file_path, 'w') as f:
        f.write(new_content)

    print(f"Version updated to: {new_version}")
else:
    print("Error: Version line not found in config.go")
