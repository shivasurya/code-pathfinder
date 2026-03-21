from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class IOModule(QueryType):
    fqns = ["io"]

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
    id="PYTHON-DJANGO-SEC-040",
    name="Django Path Traversal via open()",
    severity="HIGH",
    category="django",
    cwe="CWE-22",
    tags="python,django,path-traversal,open,owasp-a01,cwe-22",
    message="User input flows to open(). Validate file paths with os.path.realpath().",
    owasp="A01:2021",
)
def detect_django_path_traversal_open():
    """Detects Django request data flowing to open() file operations."""
    return flows(
        from_sources=_DJANGO_SOURCES,
        to_sinks=[
            IOModule.method("open").tracks(0),
            calls("open"),
        ],
        sanitized_by=[
            calls("os.path.realpath"),
            calls("os.path.abspath"),
            calls("os.path.basename"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
