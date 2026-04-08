import tarfile
import os

# SEC-110: Unsafe tarfile.extractall() — path traversal via crafted archive members

# 1. Basic extractall from untrusted archive
def extract_uploaded_archive(upload_path):
    tf = tarfile.open(upload_path, "r:gz")
    tf.extractall(path="/tmp/uploads")
    tf.close()

# 2. Context manager with extractall
def extract_with_context(archive_path, dest_dir):
    with tarfile.open(archive_path) as tar:
        tar.extractall(dest_dir)

# 3. TarFile constructor with extractall
def extract_via_constructor(path):
    tar = tarfile.TarFile(path)
    tar.extractall("/opt/data")
    tar.close()

# 4. Single member extract without validation
def extract_single_member(archive_path, member_name):
    with tarfile.open(archive_path) as tar:
        tar.extract(member_name, path="/var/lib/app")

# 5. Extractall in a loop processing multiple archives
def batch_extract(archive_list):
    for archive in archive_list:
        with tarfile.open(archive, "r:*") as tar:
            tar.extractall(path=os.path.join("/data", os.path.basename(archive)))

# 6. Extract with no path argument (extracts to cwd)
def extract_to_cwd(archive_path):
    tar = tarfile.open(archive_path)
    tar.extractall()
    tar.close()
