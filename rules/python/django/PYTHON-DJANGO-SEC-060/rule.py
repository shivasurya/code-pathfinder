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
    id="PYTHON-DJANGO-SEC-060",
    name="Django XSS in HTML Email Body",
    severity="MEDIUM",
    category="django",
    cwe="CWE-79",
    tags="python,django,xss,email,html,OWASP-A03,CWE-79",
    message="User input in HTML email body. Sanitize content before sending.",
    owasp="A03:2021",
)
def detect_django_email_xss():
    """Detects user input flowing into EmailMessage body."""
    return flows(
        from_sources=_DJANGO_SOURCES,
        to_sinks=[
            calls("EmailMessage"),
            calls("django.core.mail.EmailMessage"),
        ],
        sanitized_by=[
            calls("escape"),
            calls("strip_tags"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
