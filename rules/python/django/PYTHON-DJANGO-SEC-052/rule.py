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
    id="PYTHON-DJANGO-SEC-052",
    name="Django html_safe() Usage (Audit)",
    severity="MEDIUM",
    category="django",
    cwe="CWE-79",
    tags="python,django,xss,html-safe,audit,cwe-79",
    message="html_safe() marks output as safe HTML. Ensure content is properly sanitized.",
    owasp="A03:2021",
)
def detect_django_html_safe():
    """Audit: detects html_safe() usage."""
    return DjangoSafeString.method("html_safe")
