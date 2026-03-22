from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class DjangoSafeString(QueryType):
    fqns = ["django.utils.safestring", "django.utils.html"]

_DJANGO_SOURCES = [
    calls("request.GET.get"),
    calls("request.POST.get"),
    calls("request.GET"),
    calls("request.POST"),
    calls("request.COOKIES.get"),
    calls("request.FILES.get"),
    calls("*.GET.get"),
    calls("*.POST.get"),
]


@python_rule(
    id="PYTHON-DJANGO-SEC-051",
    name="Django mark_safe() Usage (Audit)",
    severity="MEDIUM",
    category="django",
    cwe="CWE-79",
    tags="python,django,xss,mark-safe,audit,CWE-79",
    message="mark_safe() bypasses Django auto-escaping. Ensure input is properly sanitized.",
    owasp="A03:2021",
)
def detect_django_mark_safe():
    """Audit: detects mark_safe() usage that bypasses auto-escaping."""
    return DjangoSafeString.method("mark_safe")
