import tarfile
import os

# Safe: extractall with filter='data' (Python 3.12+ safe extraction)
def safe_extract_with_filter(archive_path, dest):
    with tarfile.open(archive_path) as tar:
        tar.extractall(path=dest, filter="data")

# Safe: Only reading member names without extracting
def list_archive_contents(archive_path):
    with tarfile.open(archive_path) as tar:
        names = tar.getnames()
    return names

# Safe: Using extractfile to read a file object without writing to disk
def read_member_contents(archive_path, member_name):
    with tarfile.open(archive_path) as tar:
        f = tar.extractfile(member_name)
        if f is not None:
            return f.read()
    return None

# Safe: Validating members before extraction
def safe_extract_validated(archive_path, dest):
    with tarfile.open(archive_path) as tar:
        validated_members = []
        for member in tar.getmembers():
            member_path = os.path.realpath(os.path.join(dest, member.name))
            if member_path.startswith(os.path.realpath(dest)):
                validated_members.append(member)
        tar.extractall(path=dest, members=validated_members, filter="data")

# Safe: Using getmembers() to inspect archive metadata
def inspect_archive(archive_path):
    with tarfile.open(archive_path) as tar:
        for member in tar.getmembers():
            print(f"{member.name}: {member.size} bytes")
