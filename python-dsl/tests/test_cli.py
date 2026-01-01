"""Tests for CLI wrapper module."""
import pytest
import sys
from pathlib import Path
from unittest.mock import patch, MagicMock

from codepathfinder.cli import get_binary_path, get_binary_name, _get_platform_string


class TestGetBinaryName:
    """Tests for get_binary_name function."""

    def test_unix_binary_name(self):
        """Test binary name on Unix systems."""
        with patch.object(sys, 'platform', 'linux'):
            assert get_binary_name() == "pathfinder"

    def test_windows_binary_name(self):
        """Test binary name on Windows."""
        with patch.object(sys, 'platform', 'win32'):
            assert get_binary_name() == "pathfinder.exe"


class TestGetPlatformString:
    """Tests for platform detection."""

    @pytest.mark.parametrize("system,machine,expected", [
        ("Linux", "x86_64", "linux-amd64"),
        ("Linux", "aarch64", "linux-arm64"),
        ("Darwin", "arm64", "darwin-arm64"),
        ("Darwin", "x86_64", "darwin-amd64"),
        ("Windows", "AMD64", "windows-amd64"),
    ])
    def test_platform_detection(self, system, machine, expected):
        """Test platform string generation for various OS/arch combinations."""
        with patch('platform.system', return_value=system):
            with patch('platform.machine', return_value=machine):
                with patch('codepathfinder.cli._is_musl', return_value=False):
                    assert _get_platform_string() == expected

    def test_musl_detection(self):
        """Test musl libc detection for Alpine Linux."""
        with patch('platform.system', return_value='Linux'):
            with patch('platform.machine', return_value='x86_64'):
                with patch('codepathfinder.cli._is_musl', return_value=True):
                    assert _get_platform_string() == "linux-amd64-musl"

    def test_unsupported_architecture(self):
        """Test error handling for unsupported architectures."""
        with patch('platform.system', return_value='Linux'):
            with patch('platform.machine', return_value='mips'):
                with pytest.raises(RuntimeError, match="Unsupported architecture"):
                    _get_platform_string()

    def test_unsupported_os(self):
        """Test error handling for unsupported operating systems."""
        with patch('platform.system', return_value='FreeBSD'):
            with patch('platform.machine', return_value='x86_64'):
                with pytest.raises(RuntimeError, match="Unsupported operating system"):
                    _get_platform_string()
