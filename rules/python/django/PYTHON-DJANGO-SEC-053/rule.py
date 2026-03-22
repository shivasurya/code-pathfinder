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
    id="PYTHON-DJANGO-SEC-053",
    name="Django SafeString Subclass (Audit)",
    severity="MEDIUM",
    category="django",
    cwe="CWE-79",
    tags="python,django,xss,safestring,audit,CWE-79",
    message="Class extends SafeString/SafeData. Content bypasses auto-escaping.",
    owasp="A03:2021",
)
def detect_django_safestring_subclass():
    """Audit: detects SafeString/SafeText subclass usage."""
    return DjangoSafeString.method("SafeString")
