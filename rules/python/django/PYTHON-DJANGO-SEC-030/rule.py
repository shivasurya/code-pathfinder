from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class RequestsLib(QueryType):
    fqns = ["requests"]

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
    id="PYTHON-DJANGO-SEC-030",
    name="Django SSRF via requests Library",
    severity="HIGH",
    category="django",
    cwe="CWE-918",
    tags="python,django,ssrf,requests,owasp-a10,cwe-918",
    message="User input flows to requests HTTP call. Validate and restrict URLs.",
    owasp="A10:2021",
)
def detect_django_ssrf_requests():
    """Detects Django request data flowing to requests library calls."""
    return flows(
        from_sources=_DJANGO_SOURCES,
        to_sinks=[
            RequestsLib.method("get", "post", "put", "delete", "patch",
                               "head", "options", "request").tracks(0),
        ],
        sanitized_by=[
            calls("urllib.parse.urlparse"),
            calls("validators.url"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
