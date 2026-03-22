from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class UrllibModule(QueryType):
    fqns = ["urllib.request"]

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
    id="PYTHON-DJANGO-SEC-031",
    name="Django SSRF via urllib",
    severity="HIGH",
    category="django",
    cwe="CWE-918",
    tags="python,django,ssrf,urllib,OWASP-A10,CWE-918",
    message="User input flows to urllib.request.urlopen(). Validate and restrict URLs.",
    owasp="A10:2021",
)
def detect_django_ssrf_urllib():
    """Detects Django request data flowing to urllib calls."""
    return flows(
        from_sources=_DJANGO_SOURCES,
        to_sinks=[
            UrllibModule.method("urlopen", "Request").tracks(0),
        ],
        sanitized_by=[
            calls("urllib.parse.urlparse"),
            calls("validators.url"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
