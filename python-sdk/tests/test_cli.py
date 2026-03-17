"""Tests for CLI wrapper module."""

import os
import sys

import pytest
from unittest.mock import patch, MagicMock

from codepathfinder.cli import (
    get_binary_name,
    get_binary_path,
    main,
    _ENV_OVERRIDE,
)

# ---------------------------------------------------------------------------
# get_binary_name
# ---------------------------------------------------------------------------


class TestGetBinaryName:

    def test_unix(self):
        with patch.object(sys, "platform", "linux"):
            assert get_binary_name() == "pathfinder"

    def test_macos(self):
        with patch.object(sys, "platform", "darwin"):
            assert get_binary_name() == "pathfinder"

    def test_windows(self):
        with patch.object(sys, "platform", "win32"):
            assert get_binary_name() == "pathfinder.exe"


# ---------------------------------------------------------------------------
# get_binary_path — env var override
# ---------------------------------------------------------------------------


class TestEnvOverride:

    def test_valid_env_binary(self, tmp_path):
        binary = tmp_path / "pathfinder"
        binary.write_bytes(b"\x7fELF fake binary")
        binary.chmod(0o755)

        with patch.dict(os.environ, {_ENV_OVERRIDE: str(binary)}):
            assert get_binary_path() == binary.resolve()

    def test_env_empty_string(self):
        with patch.dict(os.environ, {_ENV_OVERRIDE: ""}):
            with pytest.raises(FileNotFoundError, match="set but empty"):
                get_binary_path()

    def test_env_nonexistent_path(self):
        with patch.dict(os.environ, {_ENV_OVERRIDE: "/no/such/binary"}):
            with pytest.raises(FileNotFoundError, match="does not exist"):
                get_binary_path()

    def test_env_points_to_directory(self, tmp_path):
        with patch.dict(os.environ, {_ENV_OVERRIDE: str(tmp_path)}):
            with pytest.raises(FileNotFoundError, match="not a regular file"):
                get_binary_path()

    def test_env_not_executable(self, tmp_path):
        binary = tmp_path / "pathfinder"
        binary.write_bytes(b"\x7fELF fake binary")
        binary.chmod(0o644)

        with patch.dict(os.environ, {_ENV_OVERRIDE: str(binary)}):
            with pytest.raises(PermissionError, match="not executable"):
                get_binary_path()

    def test_env_takes_priority_over_bundled(self, tmp_path):
        """Even if bundled binary exists, env var wins."""
        env_binary = tmp_path / "env" / "pathfinder"
        env_binary.parent.mkdir()
        env_binary.write_bytes(b"\x7fELF env")
        env_binary.chmod(0o755)

        # Set up a fake bundled binary too
        pkg_dir = tmp_path / "codepathfinder"
        cli_dir = pkg_dir / "cli"
        cli_dir.mkdir(parents=True)
        bin_dir = pkg_dir / "bin"
        bin_dir.mkdir()
        bundled = bin_dir / "pathfinder"
        bundled.write_bytes(b"\x7fELF bundled")
        bundled.chmod(0o755)

        with patch.dict(os.environ, {_ENV_OVERRIDE: str(env_binary)}):
            result = get_binary_path()
            assert result == env_binary.resolve()


# ---------------------------------------------------------------------------
# get_binary_path — bundled binary
# ---------------------------------------------------------------------------


class TestBundledBinary:

    def _setup_package(self, tmp_path, create_binary=True, executable=True):
        """Create a fake package layout and return the expected binary path."""
        pkg_dir = tmp_path / "codepathfinder"
        cli_dir = pkg_dir / "cli"
        cli_dir.mkdir(parents=True)
        bin_dir = pkg_dir / "bin"
        bin_dir.mkdir()

        binary = bin_dir / "pathfinder"
        if create_binary:
            binary.write_bytes(b"\x7fELF fake")
            binary.chmod(0o755 if executable else 0o644)

        return cli_dir, binary

    def test_bundled_found(self, tmp_path):
        cli_dir, binary = self._setup_package(tmp_path)

        with patch.dict(os.environ, {}, clear=False):
            # Make sure env override is not set
            os.environ.pop(_ENV_OVERRIDE, None)
            with patch("codepathfinder.cli.__file__", str(cli_dir / "__init__.py")):
                with patch.object(sys, "platform", "linux"):
                    assert get_binary_path() == binary

    def test_bundled_missing(self, tmp_path):
        cli_dir, _ = self._setup_package(tmp_path, create_binary=False)

        with patch.dict(os.environ, {}, clear=False):
            os.environ.pop(_ENV_OVERRIDE, None)
            with patch("codepathfinder.cli.__file__", str(cli_dir / "__init__.py")):
                with patch.object(sys, "platform", "linux"):
                    with pytest.raises(FileNotFoundError, match="not found"):
                        get_binary_path()

    def test_bundled_not_executable(self, tmp_path):
        cli_dir, _ = self._setup_package(tmp_path, executable=False)

        with patch.dict(os.environ, {}, clear=False):
            os.environ.pop(_ENV_OVERRIDE, None)
            with patch("codepathfinder.cli.__file__", str(cli_dir / "__init__.py")):
                with patch.object(sys, "platform", "linux"):
                    with pytest.raises(PermissionError, match="not executable"):
                        get_binary_path()


# ---------------------------------------------------------------------------
# main
# ---------------------------------------------------------------------------


class TestMain:

    def test_success(self, tmp_path):
        binary = tmp_path / "pathfinder"
        mock_result = MagicMock(returncode=0)

        with patch("codepathfinder.cli.get_binary_path", return_value=binary):
            with patch("subprocess.run", return_value=mock_result) as mock_run:
                with patch.object(sys, "argv", ["pathfinder", "--help"]):
                    with pytest.raises(SystemExit) as exc:
                        main()
                    assert exc.value.code == 0
                    mock_run.assert_called_once_with([str(binary), "--help"])

    def test_nonzero_exit(self, tmp_path):
        binary = tmp_path / "pathfinder"
        mock_result = MagicMock(returncode=1)

        with patch("codepathfinder.cli.get_binary_path", return_value=binary):
            with patch("subprocess.run", return_value=mock_result):
                with patch.object(sys, "argv", ["pathfinder", "scan"]):
                    with pytest.raises(SystemExit) as exc:
                        main()
                    assert exc.value.code == 1

    def test_binary_not_found(self):
        with patch(
            "codepathfinder.cli.get_binary_path",
            side_effect=FileNotFoundError("not found"),
        ):
            with pytest.raises(SystemExit) as exc:
                main()
            assert exc.value.code == 2

    def test_permission_error(self):
        with patch(
            "codepathfinder.cli.get_binary_path",
            side_effect=PermissionError("not executable"),
        ):
            with pytest.raises(SystemExit) as exc:
                main()
            assert exc.value.code == 2

    def test_os_error_on_exec(self, tmp_path):
        binary = tmp_path / "pathfinder"

        with patch("codepathfinder.cli.get_binary_path", return_value=binary):
            with patch("subprocess.run", side_effect=OSError("exec format error")):
                with patch.object(sys, "argv", ["pathfinder"]):
                    with pytest.raises(SystemExit) as exc:
                        main()
                    assert exc.value.code == 2

    def test_keyboard_interrupt(self, tmp_path):
        binary = tmp_path / "pathfinder"

        with patch("codepathfinder.cli.get_binary_path", return_value=binary):
            with patch("subprocess.run", side_effect=KeyboardInterrupt):
                with patch.object(sys, "argv", ["pathfinder"]):
                    with pytest.raises(SystemExit) as exc:
                        main()
                    assert exc.value.code == 130

    def test_arguments_forwarded(self, tmp_path):
        binary = tmp_path / "pathfinder"
        mock_result = MagicMock(returncode=0)

        with patch("codepathfinder.cli.get_binary_path", return_value=binary):
            with patch("subprocess.run", return_value=mock_result) as mock_run:
                with patch.object(
                    sys, "argv", ["pathfinder", "query", "--project", "/tmp"]
                ):
                    with pytest.raises(SystemExit):
                        main()
                    mock_run.assert_called_once_with(
                        [str(binary), "query", "--project", "/tmp"]
                    )
