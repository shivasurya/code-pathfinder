"""CLI wrapper for pathfinder binary.

This module provides the entry point for the `pathfinder` command.
It locates and executes the bundled Go binary, passing through all arguments.
"""

import os
import sys
import subprocess
import platform
import urllib.request
import tarfile
import zipfile
import tempfile
from pathlib import Path

from codepathfinder import __version__


def get_binary_name() -> str:
    """Get the binary name for the current platform."""
    if sys.platform == "win32":
        return "pathfinder.exe"
    return "pathfinder"


def get_binary_path() -> Path:
    """Get path to the pathfinder binary.

    Priority:
    1. Bundled binary in package (platform wheels)
    2. Binary in PATH (for development or manual install)
    3. Download on first use (fallback for source installs)

    Returns:
        Path to the executable binary

    Raises:
        RuntimeError: If binary cannot be found or downloaded
    """
    binary_name = get_binary_name()

    # 1. Check bundled binary (primary - from platform wheel)
    package_dir = Path(__file__).parent.parent
    bin_dir = package_dir / "bin"
    bundled_binary = bin_dir / binary_name

    if bundled_binary.exists() and os.access(bundled_binary, os.X_OK):
        return bundled_binary

    # 2. Check PATH (development mode or manual install)
    import shutil

    path_binary = shutil.which("pathfinder")
    if path_binary:
        return Path(path_binary)

    # 3. Download on first use (source distribution fallback)
    return _download_binary(bin_dir, binary_name)


def _is_musl() -> bool:
    """Detect if running on musl libc (Alpine Linux, etc.)."""
    try:
        result = subprocess.run(["ldd", "--version"], capture_output=True, text=True)
        return "musl" in result.stderr.lower() or "musl" in result.stdout.lower()
    except Exception:
        try:
            with open("/etc/os-release") as f:
                content = f.read().lower()
                return "alpine" in content
        except Exception:
            return False


def _get_platform_string() -> str:
    """Get platform string for binary download.

    Supports 96%+ of worldwide architectures:
    - Linux glibc: x86_64, aarch64
    - Linux musl: x86_64, aarch64 (Alpine Docker)
    - macOS: arm64 (M1/M2/M3), x86_64 (Intel)
    - Windows: x86_64
    """
    system = platform.system().lower()
    machine = platform.machine().lower()

    arch_map = {
        "x86_64": "amd64",
        "amd64": "amd64",
        "aarch64": "arm64",
        "arm64": "arm64",
        "armv8l": "arm64",
    }

    arch = arch_map.get(machine)
    if not arch:
        raise RuntimeError(
            f"Unsupported architecture: {machine}\n"
            f"Supported: x86_64, aarch64/arm64\n"
            f"Download manually from: https://github.com/shivasurya/code-pathfinder/releases"
        )

    os_map = {
        "linux": "linux",
        "darwin": "darwin",
        "windows": "windows",
    }

    os_name = os_map.get(system)
    if not os_name:
        raise RuntimeError(
            f"Unsupported operating system: {system}\n"
            f"Supported: Linux, macOS, Windows\n"
            f"Download manually from: https://github.com/shivasurya/code-pathfinder/releases"
        )

    if os_name == "linux" and _is_musl():
        return f"{os_name}-{arch}-musl"

    return f"{os_name}-{arch}"


def _download_binary(bin_dir: Path, binary_name: str) -> Path:
    """Download binary for current platform from GitHub releases.

    Args:
        bin_dir: Directory to store the binary
        binary_name: Name of the binary file

    Returns:
        Path to the downloaded binary

    Raises:
        RuntimeError: If download fails
    """
    platform_str = _get_platform_string()

    if sys.platform == "win32":
        archive_ext = ".zip"
    else:
        archive_ext = ".tar.gz"

    url = (
        f"https://github.com/shivasurya/code-pathfinder/releases/download/"
        f"v{__version__}/pathfinder-{platform_str}{archive_ext}"
    )

    print(f"Downloading pathfinder binary for {platform_str}...", file=sys.stderr)

    bin_dir.mkdir(parents=True, exist_ok=True)

    try:
        with tempfile.NamedTemporaryFile(suffix=archive_ext, delete=False) as tmp:
            urllib.request.urlretrieve(url, tmp.name)

            if archive_ext == ".tar.gz":
                with tarfile.open(tmp.name, "r:gz") as tar:
                    for member in tar.getmembers():
                        if member.name == "pathfinder" or member.name.endswith(
                            "/pathfinder"
                        ):
                            member.name = binary_name
                            tar.extract(member, bin_dir)
                            break
            else:
                with zipfile.ZipFile(tmp.name, "r") as zip_ref:
                    for name in zip_ref.namelist():
                        if name == "pathfinder.exe" or name.endswith("/pathfinder.exe"):
                            with zip_ref.open(name) as src:
                                (bin_dir / binary_name).write_bytes(src.read())
                            break

            os.unlink(tmp.name)
    except Exception as e:
        raise RuntimeError(
            f"Failed to download pathfinder binary from {url}: {e}\n"
            f"You can manually download from: "
            f"https://github.com/shivasurya/code-pathfinder/releases"
        ) from e

    binary_path = bin_dir / binary_name

    if sys.platform != "win32":
        os.chmod(binary_path, 0o755)

    print(f"Binary installed to: {binary_path}", file=sys.stderr)
    return binary_path


def main():
    """Entry point - execute pathfinder binary with all arguments."""
    try:
        binary = get_binary_path()
    except RuntimeError as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(2)

    result = subprocess.run([str(binary)] + sys.argv[1:])
    sys.exit(result.returncode)


if __name__ == "__main__":
    main()
