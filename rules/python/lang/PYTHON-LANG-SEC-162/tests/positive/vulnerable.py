import os
import shutil

# SEC-162: User-controlled path flows to file op without symlink resolution

# 1. os.path.join → open (classic repo file read)
def read_repo_file(repo_dir, filename):
    path = os.path.join(repo_dir, filename)
    f = open(path, "r")
    return f.read()

# 2. os.path.join → os.readlink (Weblate CVE pattern)
def get_link_target(repo_dir, name):
    path = os.path.join(repo_dir, name)
    target = os.readlink(path)
    return target

# 3. request.args.get → open (direct web input)
def download_file(request):
    filename = request.args.get("file")
    f = open(filename, "rb")
    return f.read()
