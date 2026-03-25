import zipfile
import os

# Safe: using namelist() to inspect without extracting
with zipfile.ZipFile("archive.zip") as zf:
    names = zf.namelist()

# Safe: using read() to get contents without writing to disk
with zipfile.ZipFile("archive.zip") as zf:
    data = zf.read("config.json")

# Safe: extractall with validated members list
def safe_extract(zip_path, dest_dir):
    with zipfile.ZipFile(zip_path) as zf:
        safe_members = []
        for member in zf.infolist():
            member_path = os.path.realpath(os.path.join(dest_dir, member.filename))
            if member_path.startswith(os.path.realpath(dest_dir)):
                safe_members.append(member)
        # Only extract validated members
        # zf.extractall(dest_dir, members=safe_members)  # not called here

# Safe: open() for reading specific entry
with zipfile.ZipFile("archive.zip") as zf:
    with zf.open("data.csv") as f:
        content = f.read()
