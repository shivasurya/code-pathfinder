from codepathfinder.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class Builtins(QueryType):
    fqns = ["builtins"]

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
    id="PYTHON-DJANGO-SEC-021",
    name="Django Code Injection via exec()",
    severity="CRITICAL",
    category="django",
    cwe="CWE-95",
    tags="python,django,code-injection,exec,OWASP-A03,CWE-95",
    message="User input flows to exec(). Never use exec() with untrusted data.",
    owasp="A03:2021",
)
def detect_django_exec_injection():
    """Detects Django request data flowing to exec()."""
    return flows(
        from_sources=_DJANGO_SOURCES,
        to_sinks=[
            Builtins.method("exec").tracks(0),
            calls("exec"),
        ],
        sanitized_by=[],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
