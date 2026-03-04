"""
Framework-specific presets for source/sink/sanitizer patterns.

Provides targeted matchers for Django, Flask, and FastAPI.

Example:
    from codepathfinder.frameworks import django
    from codepathfinder import flows

    flows(
        from_sources=django.sources.request_data(),
        to_sinks=django.sinks.raw_sql(),
    )
"""

from . import django, flask, fastapi

__all__ = ["django", "flask", "fastapi"]
