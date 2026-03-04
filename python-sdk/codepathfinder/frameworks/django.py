"""
Django framework presets — sources, sinks, and sanitizers.

Type names match Django 4.x/5.x class names:
- django.http.HttpRequest
- django.http.QueryDict
- django.contrib.sessions.backends.base.SessionBase
- django.db.models.QuerySet
- django.template.Template

Example:
    from codepathfinder.frameworks import django
    from codepathfinder import flows

    flows(
        from_sources=django.sources.request_data(),
        to_sinks=django.sinks.raw_sql(),
        sanitized_by=django.sanitizers.orm_parameterization(),
    )
"""

from ..matchers import calls, calls_on
from ..logic import Or


class DjangoSources:
    """Django request input sources."""

    @staticmethod
    def request_data():
        """All Django request input — GET, POST, body, META.

        Covers HttpRequest.GET, HttpRequest.POST, QueryDict access,
        HttpRequest.body, HttpRequest.META.
        """
        return Or(
            calls_on("HttpRequest", "GET", fallback="name"),
            calls_on("HttpRequest", "POST", fallback="name"),
            calls_on("QueryDict", "__getitem__", fallback="name"),
            calls_on("QueryDict", "get", fallback="name"),
            calls_on("HttpRequest", "body", fallback="name"),
            calls_on("HttpRequest", "META", fallback="name"),
        )

    @staticmethod
    def session_data():
        """Django session data access."""
        return calls_on("SessionBase", "__getitem__", fallback="name")

    @staticmethod
    def url_params():
        """Django URL resolver keyword arguments."""
        return calls("*kwargs*")


class DjangoSinks:
    """Django dangerous operation sinks."""

    @staticmethod
    def raw_sql():
        """Raw SQL execution — type-aware, strict fallback.

        Covers Cursor.execute, RawSQL(), QuerySet.raw, QuerySet.extra.
        """
        return Or(
            calls_on("Cursor", "execute", fallback="none"),
            calls("RawSQL"),
            calls_on("QuerySet", "raw", fallback="none"),
            calls_on("QuerySet", "extra", fallback="none"),
        )

    @staticmethod
    def template_render():
        """Template rendering sinks (XSS risk).

        Covers render_to_string, Template.render, mark_safe.
        """
        return Or(
            calls("render_to_string"),
            calls_on("Template", "render", fallback="name"),
            calls("mark_safe"),
        )

    @staticmethod
    def redirect():
        """HTTP redirect sink (open redirect risk)."""
        return calls("HttpResponseRedirect")


class DjangoSanitizers:
    """Django sanitization functions."""

    @staticmethod
    def orm_parameterization():
        """ORM parameterized queries via QuerySet.filter."""
        return calls_on("QuerySet", "filter", fallback="name")

    @staticmethod
    def escape():
        """Django HTML escaping."""
        return calls("django.utils.html.escape")


sources = DjangoSources
sinks = DjangoSinks
sanitizers = DjangoSanitizers
