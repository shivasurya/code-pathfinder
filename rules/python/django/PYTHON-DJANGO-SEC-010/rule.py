from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class OSModule(QueryType):
    fqns = ["os"]

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
    id="PYTHON-DJANGO-SEC-010",
    name="Django Command Injection via os.system()",
    severity="CRITICAL",
    category="django",
    cwe="CWE-78",
    tags="python,django,command-injection,os-system,OWASP-A03,CWE-78",
    message="User input flows to os.system(). Use subprocess with list args and shlex.quote().",
    owasp="A03:2021",
)
def detect_django_os_system_injection():
    """Detects Django request data flowing to os.system()/os.popen()."""
    return flows(
        from_sources=_DJANGO_SOURCES,
        to_sinks=[
            OSModule.method("system", "popen", "popen2", "popen3", "popen4"),
        ],
        sanitized_by=[
            calls("shlex.quote"),
            calls("shlex.split"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
