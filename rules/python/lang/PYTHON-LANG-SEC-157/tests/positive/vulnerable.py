import zipfile

# SEC-157: ZipFile.extract() without path validation
zf = zipfile.ZipFile("archive.zip")
zf.extract("malicious/../../etc/passwd", "/tmp/output")

# SEC-157: ZipFile.extractall() without member validation
with zipfile.ZipFile("upload.zip") as zf:
    zf.extractall("/tmp/output")

# SEC-157: extract in a loop without checking for path traversal
with zipfile.ZipFile(uploaded_file) as zf:
    for member in zf.namelist():
        zf.extract(member, dest_dir)

# SEC-157: ZipFile constructed inline and extracted
zipfile.ZipFile(user_upload).extractall(target_path)

# SEC-157: extract with info object
with zipfile.ZipFile("data.zip") as zf:
    for info in zf.infolist():
        zf.extract(info, output_dir)
