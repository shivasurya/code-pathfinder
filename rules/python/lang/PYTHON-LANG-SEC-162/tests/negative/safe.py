import os
from pathlib import Path

# Safe: os.path.realpath() resolves symlinks (not a symlink operation itself)
real = os.path.realpath(user_path)
if real.startswith(ALLOWED_DIR):
    with open(real) as f:
        data = f.read()

# Safe: Path.resolve() with is_relative_to() check (no symlink API call)
p = Path(user_input).resolve()
if p.is_relative_to(Path("/app/data")):
    content = p.read_text()

# Safe: stat with follow_symlinks=False (does not follow symlinks)
info = os.stat(path, follow_symlinks=False)

# Safe: using os.open with O_NOFOLLOW flag (rejects symlinks)
fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW)

# Safe: checking file type without symlink operations
import stat
mode = os.lstat(path).st_mode
is_regular = stat.S_ISREG(mode)
