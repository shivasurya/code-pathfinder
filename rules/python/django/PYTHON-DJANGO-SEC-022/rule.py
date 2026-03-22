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
    id="PYTHON-DJANGO-SEC-022",
    name="Django Globals Misuse Code Execution",
    severity="HIGH",
    category="django",
    cwe="CWE-94",
    tags="python,django,code-injection,globals,owasp-a03,cwe-94",
    message="User input used to index globals(). This allows arbitrary code execution.",
    owasp="A03:2021",
)
def detect_django_globals_misuse():
    """Detects Django request data used with globals() for code execution."""
    return flows(
        from_sources=_DJANGO_SOURCES,
        to_sinks=[
            calls("globals"),
            calls("globals().get"),
        ],
        sanitized_by=[],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
