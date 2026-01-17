"""Tests for CLI wrapper module."""

import pytest
import sys
from unittest.mock import patch, MagicMock, mock_open

from codepathfinder.cli import (
    get_binary_path,
    get_binary_name,
    _get_platform_string,
    _is_musl,
    _download_binary,
    main,
)


class TestGetBinaryName:
    """Tests for get_binary_name function."""

    def test_unix_binary_name(self):
        """Test binary name on Unix systems."""
        with patch.object(sys, "platform", "linux"):
            assert get_binary_name() == "pathfinder"

    def test_windows_binary_name(self):
        """Test binary name on Windows."""
        with patch.object(sys, "platform", "win32"):
            assert get_binary_name() == "pathfinder.exe"


class TestGetPlatformString:
    """Tests for platform detection."""

    @pytest.mark.parametrize(
        "system,machine,expected",
        [
            ("Linux", "x86_64", "linux-amd64"),
            ("Linux", "aarch64", "linux-arm64"),
            ("Darwin", "arm64", "darwin-arm64"),
            ("Darwin", "x86_64", "darwin-amd64"),
            ("Windows", "AMD64", "windows-amd64"),
        ],
    )
    def test_platform_detection(self, system, machine, expected):
        """Test platform string generation for various OS/arch combinations."""
        with patch("platform.system", return_value=system):
            with patch("platform.machine", return_value=machine):
                with patch("codepathfinder.cli._is_musl", return_value=False):
                    assert _get_platform_string() == expected

    def test_musl_detection(self):
        """Test musl libc detection for Alpine Linux."""
        with patch("platform.system", return_value="Linux"):
            with patch("platform.machine", return_value="x86_64"):
                with patch("codepathfinder.cli._is_musl", return_value=True):
                    assert _get_platform_string() == "linux-amd64-musl"

    def test_unsupported_architecture(self):
        """Test error handling for unsupported architectures."""
        with patch("platform.system", return_value="Linux"):
            with patch("platform.machine", return_value="mips"):
                with pytest.raises(RuntimeError, match="Unsupported architecture"):
                    _get_platform_string()

    def test_unsupported_os(self):
        """Test error handling for unsupported operating systems."""
        with patch("platform.system", return_value="FreeBSD"):
            with patch("platform.machine", return_value="x86_64"):
                with pytest.raises(RuntimeError, match="Unsupported operating system"):
                    _get_platform_string()


class TestIsMusl:
    """Tests for _is_musl function."""

    def test_musl_via_ldd_stderr(self):
        """Test musl detection via ldd stderr output."""
        mock_result = MagicMock()
        mock_result.stderr = "musl libc"
        mock_result.stdout = ""
        with patch("subprocess.run", return_value=mock_result):
            assert _is_musl() is True

    def test_musl_via_ldd_stdout(self):
        """Test musl detection via ldd stdout output."""
        mock_result = MagicMock()
        mock_result.stderr = ""
        mock_result.stdout = "musl libc"
        with patch("subprocess.run", return_value=mock_result):
            assert _is_musl() is True

    def test_musl_via_os_release(self):
        """Test musl detection via /etc/os-release (Alpine)."""
        # Make ldd fail so it falls back to os-release
        with patch("subprocess.run", side_effect=Exception("ldd not found")):
            with patch("builtins.open", mock_open(read_data="NAME=Alpine Linux")):
                assert _is_musl() is True

    def test_not_musl(self):
        """Test detection when not on musl system."""
        mock_result = MagicMock()
        mock_result.stderr = "glibc"
        mock_result.stdout = "glibc"
        with patch("subprocess.run", return_value=mock_result):
            assert _is_musl() is False

    def test_musl_detection_exception_fallback(self):
        """Test fallback when both ldd and os-release fail."""
        with patch("subprocess.run", side_effect=Exception("ldd not found")):
            with patch("builtins.open", side_effect=Exception("file not found")):
                assert _is_musl() is False


class TestGetBinaryPath:
    """Tests for get_binary_path function."""

    def test_bundled_binary_exists(self, tmp_path):
        """Test priority 1: Bundled binary in package."""
        binary_name = "pathfinder"
        # Create structure: tmp_path/codepathfinder/cli/__init__.py and tmp_path/codepathfinder/bin/pathfinder
        # Path(__file__).parent.parent should resolve to codepathfinder_dir
        codepathfinder_dir = tmp_path / "codepathfinder"
        codepathfinder_dir.mkdir()
        cli_dir = codepathfinder_dir / "cli"
        cli_dir.mkdir()
        bin_dir = codepathfinder_dir / "bin"
        bin_dir.mkdir()
        binary_path = bin_dir / binary_name
        binary_path.write_text("#!/bin/sh\necho test")
        binary_path.chmod(0o755)

        # Patch __file__ to point to cli subdirectory so .parent.parent resolves correctly
        with patch("codepathfinder.cli.__file__", str(cli_dir / "__init__.py")):
            with patch.object(sys, "platform", "linux"):
                result = get_binary_path()
                assert result == binary_path

    def test_binary_in_path(self, tmp_path):
        """Test priority 2: Binary in PATH."""
        binary_path = tmp_path / "pathfinder"
        binary_path.write_text("#!/bin/sh\necho test")
        binary_path.chmod(0o755)

        with patch("codepathfinder.cli.__file__", "/nonexistent/codepathfinder/cli.py"):
            with patch("shutil.which", return_value=str(binary_path)):
                result = get_binary_path()
                assert result == binary_path

    def test_download_binary_fallback(self, tmp_path):
        """Test priority 3: Download on first use."""
        with patch("codepathfinder.cli.__file__", "/nonexistent/codepathfinder/cli.py"):
            with patch("shutil.which", return_value=None):
                with patch(
                    "codepathfinder.cli._download_binary",
                    return_value=tmp_path / "pathfinder",
                ):
                    result = get_binary_path()
                    assert result == tmp_path / "pathfinder"


class TestDownloadBinary:
    """Tests for _download_binary function."""

    def test_download_tar_gz(self, tmp_path):
        """Test downloading tar.gz archive on Unix."""
        bin_dir = tmp_path / "bin"
        binary_name = "pathfinder"

        with patch.object(sys, "platform", "linux"):
            with patch(
                "codepathfinder.cli._get_platform_string", return_value="linux-amd64"
            ):
                with patch("urllib.request.urlretrieve") as mock_retrieve:
                    with patch("tarfile.open") as mock_tarfile:
                        # Mock tar extraction
                        mock_tar = MagicMock()
                        mock_member = MagicMock()
                        mock_member.name = "pathfinder"
                        mock_tar.getmembers.return_value = [mock_member]
                        mock_tarfile.return_value.__enter__.return_value = mock_tar

                        # Create the binary after "extraction"
                        def create_binary(*args, **kwargs):
                            bin_dir.mkdir(parents=True, exist_ok=True)
                            (bin_dir / binary_name).write_text("binary")
                            (bin_dir / binary_name).chmod(0o755)

                        mock_tar.extract.side_effect = create_binary

                        result = _download_binary(bin_dir, binary_name)
                        assert result == bin_dir / binary_name
                        assert mock_retrieve.called

    def test_download_zip(self, tmp_path):
        """Test downloading zip archive on Windows."""
        bin_dir = tmp_path / "bin"
        binary_name = "pathfinder.exe"

        with patch.object(sys, "platform", "win32"):
            with patch(
                "codepathfinder.cli._get_platform_string", return_value="windows-amd64"
            ):
                with patch("urllib.request.urlretrieve") as mock_retrieve:
                    with patch("zipfile.ZipFile") as mock_zipfile:
                        # Mock zip extraction
                        mock_zip = MagicMock()
                        mock_zip.namelist.return_value = ["pathfinder.exe"]
                        mock_file = MagicMock()
                        mock_file.read.return_value = b"binary"
                        mock_zip.open.return_value.__enter__.return_value = mock_file
                        mock_zipfile.return_value.__enter__.return_value = mock_zip

                        result = _download_binary(bin_dir, binary_name)
                        assert result == bin_dir / binary_name
                        assert mock_retrieve.called

    def test_download_failure(self, tmp_path):
        """Test download failure handling."""
        bin_dir = tmp_path / "bin"
        binary_name = "pathfinder"

        with patch.object(sys, "platform", "linux"):
            with patch(
                "codepathfinder.cli._get_platform_string", return_value="linux-amd64"
            ):
                with patch(
                    "urllib.request.urlretrieve",
                    side_effect=Exception("Network error"),
                ):
                    with pytest.raises(RuntimeError, match="Failed to download"):
                        _download_binary(bin_dir, binary_name)


class TestMain:
    """Tests for main entry point."""

    def test_main_success(self, tmp_path):
        """Test main function with successful execution."""
        binary_path = tmp_path / "pathfinder"

        with patch("codepathfinder.cli.get_binary_path", return_value=binary_path):
            mock_result = MagicMock()
            mock_result.returncode = 0
            with patch("subprocess.run", return_value=mock_result) as mock_run:
                with pytest.raises(SystemExit) as exc_info:
                    with patch.object(sys, "argv", ["pathfinder", "--help"]):
                        main()
                assert exc_info.value.code == 0
                mock_run.assert_called_once_with([str(binary_path), "--help"])

    def test_main_binary_not_found(self):
        """Test main function when binary cannot be found."""
        with patch(
            "codepathfinder.cli.get_binary_path",
            side_effect=RuntimeError("Binary not found"),
        ):
            with pytest.raises(SystemExit) as exc_info:
                main()
            assert exc_info.value.code == 2

    def test_main_with_arguments(self, tmp_path):
        """Test main function passes arguments correctly."""
        binary_path = tmp_path / "pathfinder"

        with patch("codepathfinder.cli.get_binary_path", return_value=binary_path):
            mock_result = MagicMock()
            mock_result.returncode = 1
            with patch("subprocess.run", return_value=mock_result) as mock_run:
                with pytest.raises(SystemExit) as exc_info:
                    with patch.object(
                        sys, "argv", ["pathfinder", "query", "--project", "/tmp"]
                    ):
                        main()
                assert exc_info.value.code == 1
                mock_run.assert_called_once_with(
                    [str(binary_path), "query", "--project", "/tmp"]
                )
