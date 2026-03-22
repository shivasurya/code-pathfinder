from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

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
    id="PYTHON-DJANGO-SEC-070",
    name="Django Insecure Cookie Settings",
    severity="MEDIUM",
    category="django",
    cwe="CWE-614",
    tags="python,django,cookies,security,OWASP-A05,CWE-614",
    message="set_cookie() without secure/httponly flags. Set secure=True, httponly=True, samesite='Lax'.",
    owasp="A05:2021",
)
def detect_django_insecure_cookies():
    """Audit: detects set_cookie() calls that may lack security attributes."""
    return calls("*.set_cookie")
