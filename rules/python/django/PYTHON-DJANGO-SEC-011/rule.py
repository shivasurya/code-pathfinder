from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class SubprocessModule(QueryType):
    fqns = ["subprocess"]

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
    id="PYTHON-DJANGO-SEC-011",
    name="Django Command Injection via subprocess",
    severity="CRITICAL",
    category="django",
    cwe="CWE-78",
    tags="python,django,command-injection,subprocess,OWASP-A03,CWE-78",
    message="User input flows to subprocess call. Use shlex.quote() or avoid shell=True.",
    owasp="A03:2021",
)
def detect_django_subprocess_injection():
    """Detects Django request data flowing to subprocess functions."""
    return flows(
        from_sources=_DJANGO_SOURCES,
        to_sinks=[
            SubprocessModule.method("call", "check_call", "check_output",
                                    "run", "Popen", "getoutput", "getstatusoutput").tracks(0),
        ],
        sanitized_by=[
            calls("shlex.quote"),
            calls("shlex.split"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
