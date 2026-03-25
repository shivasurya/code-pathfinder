import os
from pathlib import Path

# SEC-162: os.readlink on user-controlled path
user_path = request.args.get("path")
target = os.readlink(user_path)

# SEC-162: os.symlink with user-controlled arguments
target_file = input("Enter target: ")
link_name = input("Enter link name: ")
os.symlink(target_file, link_name)

# SEC-162: Path.is_symlink() check on user-controlled path
repo_path = get_user_repo_path()
if Path(repo_path).is_symlink():
    content = open(repo_path).read()

# SEC-162: os.path.islink on user-controlled path
uploaded = request.files["file"].filename
if os.path.islink(uploaded):
    real_target = os.readlink(uploaded)

# SEC-162: readlink in a directory traversal loop
for entry in os.scandir(user_directory):
    if entry.is_symlink():
        resolved = os.readlink(entry.path)
