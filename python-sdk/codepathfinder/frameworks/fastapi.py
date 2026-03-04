"""
FastAPI framework presets — sources, sinks, and sanitizers.

Type names match FastAPI 0.100+/Starlette class names:
- fastapi.Query, fastapi.Path, fastapi.Body, fastapi.Header, fastapi.Cookie
- starlette.requests.Request
- pydantic.BaseModel

FastAPI uses dependency injection for inputs — Query(), Path(), Body()
are the primary source of user-controlled data.

Example:
    from codepathfinder.frameworks import fastapi
    from codepathfinder import flows, sinks

    flows(
        from_sources=fastapi.sources.request_params(),
        to_sinks=sinks.sql_execution(),
    )
"""

from ..matchers import calls, calls_on
from ..logic import Or


class FastAPISources:
    """FastAPI dependency injection input sources."""

    @staticmethod
    def request_params():
        """FastAPI dependency injection parameters.

        Covers Query(), Path(), Body(), Header(), Cookie().
        """
        return Or(
            calls("Query"),
            calls("Path"),
            calls("Body"),
            calls("Header"),
            calls("Cookie"),
        )

    @staticmethod
    def request_data():
        """Starlette Request object access."""
        return Or(
            calls_on("Request", "json", fallback="name"),
            calls_on("Request", "body", fallback="name"),
            calls_on("Request", "form", fallback="name"),
        )

    @staticmethod
    def form_data():
        """FastAPI Form() dependency injection."""
        return calls("Form")


class FastAPISinks:
    """FastAPI response sinks."""

    @staticmethod
    def json_response():
        """JSON response sinks (data exposure risk)."""
        return Or(
            calls("JSONResponse"),
            calls("jsonable_encoder"),
        )

    @staticmethod
    def redirect():
        """HTTP redirect sink (open redirect risk)."""
        return calls("RedirectResponse")

    @staticmethod
    def file_response():
        """File response sink (path traversal risk)."""
        return calls("FileResponse")


class FastAPISanitizers:
    """FastAPI/Pydantic sanitization."""

    @staticmethod
    def pydantic_validation():
        """Pydantic model validation as sanitizer."""
        return calls_on("BaseModel", "model_validate", fallback="name")

    @staticmethod
    def depends():
        """FastAPI Depends() for dependency injection validation."""
        return calls("Depends")


sources = FastAPISources
sinks = FastAPISinks
sanitizers = FastAPISanitizers
