# SEC-162: Sink file — tainted path reaches file operations without symlink check

import os


def process_repo_file(file_path):
    """Tainted path → open() without os.path.realpath() — symlink attack."""
    with open(file_path, "r") as f:
        return f.read()


def read_translation_file(file_path):
    """Tainted path → os.readlink() — follows attacker symlink."""
    if os.path.islink(file_path):
        target = os.readlink(file_path)
        return target
    with open(file_path, "r") as f:
        return f.read()
