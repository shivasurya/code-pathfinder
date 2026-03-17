"""CLI wrapper for the pathfinder Go binary.

This module provides the entry point for the ``pathfinder`` command
registered via ``pyproject.toml [project.scripts]``.  It locates the
bundled native binary and delegates all arguments to it.
"""

import os
import sys
import subprocess
from pathlib import Path

_ENV_OVERRIDE = "PATHFINDER_BINARY"


def get_binary_name() -> str:
    """Return the platform-specific binary filename."""
    if sys.platform == "win32":
        return "pathfinder.exe"
    return "pathfinder"


def get_binary_path() -> Path:
    """Resolve the pathfinder native binary.

    Resolution order:
        1. ``PATHFINDER_BINARY`` environment variable (dev / CI override).
        2. Bundled binary shipped inside the platform wheel
           (``<package>/bin/pathfinder``).

    Returns:
        Absolute path to the executable.

    Raises:
        FileNotFoundError: If the binary cannot be located.
        PermissionError: If the binary exists but is not executable.
    """
    # 1. Explicit env-var override (useful for local dev builds)
    env_value = os.environ.get(_ENV_OVERRIDE)
    if env_value is not None:
        return _resolve_env_binary(env_value)

    # 2. Bundled binary from the platform wheel
    return _resolve_bundled_binary()


def _resolve_env_binary(raw_path: str) -> Path:
    """Validate and return the path given by PATHFINDER_BINARY."""
    if not raw_path:
        raise FileNotFoundError(
            f"{_ENV_OVERRIDE} is set but empty. "
            f"Provide the absolute path to the pathfinder Go binary "
            f"or unset the variable."
        )

    path = Path(raw_path).resolve()

    if not path.exists():
        raise FileNotFoundError(
            f"{_ENV_OVERRIDE} points to '{path}' which does not exist."
        )

    if not path.is_file():
        raise FileNotFoundError(
            f"{_ENV_OVERRIDE} points to '{path}' which is not a regular file."
        )

    if not os.access(path, os.X_OK):
        raise PermissionError(
            f"{_ENV_OVERRIDE} points to '{path}' which is not executable.\n"
            f"Fix with: chmod +x {path}"
        )

    return path


def _resolve_bundled_binary() -> Path:
    """Locate the Go binary bundled inside the installed wheel."""
    binary_name = get_binary_name()

    # <site-packages>/codepathfinder/cli/__init__.py  →  parent.parent = codepathfinder/
    package_dir = Path(__file__).resolve().parent.parent
    bundled = package_dir / "bin" / binary_name

    if not bundled.exists():
        raise FileNotFoundError(
            f"pathfinder binary not found at {bundled}\n"
            f"This usually means the package was installed from source "
            f"(editable / sdist) instead of a platform wheel.\n"
            f"To fix:\n"
            f"  pip install --force-reinstall codepathfinder\n"
            f"Or set {_ENV_OVERRIDE} to point to a local Go build."
        )

    if not os.access(bundled, os.X_OK):
        raise PermissionError(
            f"pathfinder binary at {bundled} is not executable.\n"
            f"Fix with: chmod +x {bundled}"
        )

    return bundled


def main() -> None:
    """Entry point — resolve the binary and exec with forwarded args."""
    try:
        binary = get_binary_path()
    except (FileNotFoundError, PermissionError) as exc:
        print(f"Error: {exc}", file=sys.stderr)
        sys.exit(2)

    try:
        result = subprocess.run([str(binary)] + sys.argv[1:])
        sys.exit(result.returncode)
    except KeyboardInterrupt:
        sys.exit(130)
    except OSError as exc:
        print(f"Error: failed to execute {binary}: {exc}", file=sys.stderr)
        sys.exit(2)


if __name__ == "__main__":
    main()
