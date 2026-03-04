"""
Semantic source categories for taint analysis.

Pre-built source matchers covering common frameworks (Django, Flask, stdlib).
Each function returns a composable matcher that can be used in flows().

Example:
    from codepathfinder import sources, flows, calls

    flows(
        from_sources=sources.http_input(),
        to_sinks=calls("eval"),
    )
"""

from .matchers import calls, calls_on
from .logic import Or


def http_params():
    """HTTP query parameters and form data.

    Covers:
    - Django: HttpRequest.GET, HttpRequest.POST, QueryDict access
    - Flask: request.args, request.form, request.values
    - Generic: request.get, request.args.*
    """
    return Or(
        # Django
        calls_on("HttpRequest", "GET", fallback="name"),
        calls_on("HttpRequest", "POST", fallback="name"),
        calls_on("QueryDict", "__getitem__", fallback="name"),
        # Flask
        calls_on("Request", "args", fallback="name"),
        calls_on("Request", "form", fallback="name"),
        calls_on("Request", "values", fallback="name"),
        # Generic
        calls("request.get"),
        calls("request.args.*"),
    )


def http_body():
    """HTTP request body and JSON data.

    Covers:
    - Flask: request.json, request.data, request.get_json()
    - Django: HttpRequest.body
    """
    return Or(
        # Flask
        calls_on("Request", "json", fallback="name"),
        calls_on("Request", "data", fallback="name"),
        calls_on("Request", "get_json", fallback="name"),
        # Django
        calls_on("HttpRequest", "body", fallback="name"),
    )


def http_headers():
    """HTTP request headers.

    Covers:
    - Django: HttpRequest.META, HttpRequest.headers
    - Flask: request.headers
    """
    return Or(
        # Django
        calls_on("HttpRequest", "META", fallback="name"),
        calls_on("HttpRequest", "headers", fallback="name"),
        # Flask
        calls_on("Request", "headers", fallback="name"),
    )


def http_cookies():
    """HTTP cookies.

    Covers:
    - Django: HttpRequest.COOKIES
    - Flask: request.cookies
    """
    return Or(
        calls_on("HttpRequest", "COOKIES", fallback="name"),
        calls_on("Request", "cookies", fallback="name"),
    )


def http_input():
    """All HTTP request data sources (params, body, headers, cookies).

    Convenience function combining all HTTP-related sources.
    """
    return Or(
        http_params(),
        http_body(),
        http_headers(),
        http_cookies(),
    )


def file_read():
    """File read operations.

    Covers:
    - Built-in: open(), read(), readlines(), readline()
    - pathlib: Path.read_text(), Path.read_bytes()
    """
    return Or(
        calls("open"),
        calls("read"),
        calls("readlines"),
        calls("readline"),
        calls_on("Path", "read_text", fallback="name"),
        calls_on("Path", "read_bytes", fallback="name"),
    )


def file_path():
    """File path inputs that may be user-controlled.

    Covers:
    - os.path: join, abspath, expanduser
    - pathlib: Path()
    """
    return Or(
        calls("os.path.join"),
        calls("os.path.abspath"),
        calls("os.path.expanduser"),
        calls_on("Path", "__init__", fallback="name"),
    )


def env_vars():
    """Environment variable access.

    Covers:
    - os: getenv, environ.get, environ.*
    """
    return Or(
        calls("os.getenv"),
        calls("os.environ.get"),
        calls("os.environ.*"),
    )


def cli_args():
    """Command-line argument access.

    Covers:
    - sys.argv
    - argparse: ArgumentParser.parse_args(), parse_known_args()
    """
    return Or(
        calls("sys.argv"),
        calls_on("ArgumentParser", "parse_args", fallback="name"),
        calls_on("ArgumentParser", "parse_known_args", fallback="name"),
    )


def database_result():
    """Database query results that may contain untrusted data.

    Covers:
    - sqlite3/DB-API: Cursor.fetchone, fetchall, fetchmany
    - SQLAlchemy: Query.all, Query.first, Query.one
    """
    return Or(
        # DB-API
        calls_on("Cursor", "fetchone", fallback="name"),
        calls_on("Cursor", "fetchall", fallback="name"),
        calls_on("Cursor", "fetchmany", fallback="name"),
        # SQLAlchemy
        calls_on("Query", "all", fallback="name"),
        calls_on("Query", "first", fallback="name"),
        calls_on("Query", "one", fallback="name"),
    )


def user_input():
    """All user-controlled input sources (comprehensive).

    Combines HTTP, file, environment, and CLI sources.
    Use this for broad taint tracking.
    """
    return Or(
        http_input(),
        file_read(),
        env_vars(),
        cli_args(),
    )
