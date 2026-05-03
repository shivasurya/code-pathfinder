from codepathfinder.python_decorators import python_rule
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
    id="PYTHON-DJANGO-SEC-071",
    name="Django CSRF Exempt Decorator",
    severity="MEDIUM",
    category="django",
    cwe="CWE-352",
    tags="python,django,csrf,security,OWASP-A05,CWE-352",
    message="@csrf_exempt disables CSRF protection. Ensure this is intentional.",
    owasp="A05:2021",
)
def detect_django_csrf_exempt():
    """Audit: detects @csrf_exempt decorator usage."""
    return calls("csrf_exempt")
