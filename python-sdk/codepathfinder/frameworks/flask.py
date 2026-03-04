"""
Flask framework presets — sources, sinks, and sanitizers.

Type names match Flask 2.x/3.x and Werkzeug class names:
- flask.wrappers.Request (extends werkzeug.wrappers.Request)
- flask.templating (Jinja2)
- werkzeug.utils.send_file

Example:
    from codepathfinder.frameworks import flask
    from codepathfinder import flows

    flows(
        from_sources=flask.sources.request_data(),
        to_sinks=flask.sinks.render_template(),
        sanitized_by=flask.sanitizers.escape(),
    )
"""

from ..matchers import calls, calls_on
from ..logic import Or


class FlaskSources:
    """Flask/Werkzeug request input sources."""

    @staticmethod
    def request_data():
        """All Flask request input — args, form, json, data, files.

        Covers request.args, request.form, request.json,
        request.data, request.files, request.values.
        """
        return Or(
            calls_on("Request", "args", fallback="name"),
            calls_on("Request", "form", fallback="name"),
            calls_on("Request", "json", fallback="name"),
            calls_on("Request", "data", fallback="name"),
            calls_on("Request", "files", fallback="name"),
            calls_on("Request", "values", fallback="name"),
        )

    @staticmethod
    def headers():
        """Flask request headers."""
        return calls_on("Request", "headers", fallback="name")

    @staticmethod
    def cookies():
        """Flask request cookies."""
        return calls_on("Request", "cookies", fallback="name")


class FlaskSinks:
    """Flask dangerous operation sinks."""

    @staticmethod
    def render_template():
        """Template rendering sinks (SSTI risk).

        Covers render_template_string, render_template.
        """
        return Or(
            calls("render_template_string"),
            calls("render_template"),
        )

    @staticmethod
    def send_file():
        """File serving sink (path traversal risk)."""
        return calls("send_file")

    @staticmethod
    def redirect():
        """HTTP redirect sink (open redirect risk)."""
        return calls("redirect")


class FlaskSanitizers:
    """Flask/Jinja2 sanitization functions."""

    @staticmethod
    def escape():
        """Markup escaping via markupsafe."""
        return calls("markupsafe.escape")

    @staticmethod
    def autoescape():
        """Jinja2 autoescape (template-level sanitization)."""
        return calls("Markup")


sources = FlaskSources
sinks = FlaskSinks
sanitizers = FlaskSanitizers
