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
    id="PYTHON-DJANGO-SEC-061",
    name="Django XSS in send_mail html_message",
    severity="MEDIUM",
    category="django",
    cwe="CWE-79",
    tags="python,django,xss,send-mail,html,owasp-a03,cwe-79",
    message="User input in send_mail() html_message parameter. Sanitize content.",
    owasp="A03:2021",
)
def detect_django_sendmail_xss():
    """Detects user input flowing into send_mail() html_message."""
    return flows(
        from_sources=_DJANGO_SOURCES,
        to_sinks=[
            calls("send_mail"),
            calls("django.core.mail.send_mail"),
        ],
        sanitized_by=[
            calls("escape"),
            calls("strip_tags"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
