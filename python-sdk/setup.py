# Build configuration lives in pyproject.toml.
# This shim exists for tools that still invoke `python setup.py` directly.
from setuptools import setup

setup()
