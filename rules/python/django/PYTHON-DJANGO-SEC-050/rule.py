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
    id="PYTHON-DJANGO-SEC-050",
    name="Django Direct HttpResponse Usage",
    severity="MEDIUM",
    category="django",
    cwe="CWE-79",
    tags="python,django,xss,httpresponse,OWASP-A03,CWE-79",
    message="Direct HttpResponse with user input detected. Use templates with auto-escaping.",
    owasp="A03:2021",
)
def detect_django_httpresponse_xss():
    """Detects user input flowing to HttpResponse without escaping."""
    return flows(
        from_sources=_DJANGO_SOURCES,
        to_sinks=[
            calls("HttpResponse"),
            calls("HttpResponseBadRequest"),
            calls("HttpResponseNotFound"),
            calls("HttpResponseForbidden"),
            calls("HttpResponseServerError"),
        ],
        sanitized_by=[
            calls("escape"),
            calls("django.utils.html.escape"),
            calls("conditional_escape"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
