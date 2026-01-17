"""Setup script for codepathfinder Python SDK."""

from setuptools import setup, find_packages
from pathlib import Path

# Read version from __init__.py
version = {}
with open("codepathfinder/__init__.py") as f:
    for line in f:
        if line.startswith("__version__"):
            exec(line, version)
            break

# Read README for long description
readme_path = Path("README.md")
readme = readme_path.read_text(encoding="utf-8") if readme_path.exists() else ""

setup(
    name="codepathfinder",
    version=version.get("__version__", "1.0.0"),
    description="Python SDK for code-pathfinder static analysis for modern security teams",
    long_description=readme,
    long_description_content_type="text/markdown",
    author="code-pathfinder contributors",
    url="https://github.com/shivasurya/code-pathfinder",
    packages=find_packages(exclude=["tests", "tests.*"]),
    python_requires=">=3.8",
    license="AGPL-3.0",
    install_requires=[
        # No external dependencies (stdlib only!)
    ],
    extras_require={
        "dev": [
            "pytest>=7.0.0",
            "pytest-cov>=4.0.0",
            "black>=23.0.0",
            "mypy>=1.0.0",
            "ruff>=0.1.0",
        ],
    },
    classifiers=[
        "Development Status :: 4 - Beta",
        "Intended Audience :: Developers",
        "License :: OSI Approved :: GNU Affero General Public License v3",
        "Programming Language :: Python :: 3",
        "Programming Language :: Python :: 3.8",
        "Programming Language :: Python :: 3.9",
        "Programming Language :: Python :: 3.10",
        "Programming Language :: Python :: 3.11",
        "Programming Language :: Python :: 3.12",
        "Topic :: Security",
        "Topic :: Software Development :: Testing",
    ],
)
