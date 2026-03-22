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
    id="PYTHON-DJANGO-SEC-041",
    name="Django Path Traversal via os.path.join()",
    severity="HIGH",
    category="django",
    cwe="CWE-22",
    tags="python,django,path-traversal,os-path,OWASP-A01,CWE-22",
    message="User input flows to os.path.join() then to file operations. Validate paths.",
    owasp="A01:2021",
)
def detect_django_path_traversal_join():
    """Detects Django request data in os.path.join() reaching file operations."""
    return flows(
        from_sources=_DJANGO_SOURCES,
        to_sinks=[
            calls("open"),
            calls("os.path.join"),
        ],
        sanitized_by=[
            calls("os.path.realpath"),
            calls("os.path.abspath"),
            calls("os.path.basename"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
