from rules.python_decorators import python_rule
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
    id="PYTHON-DJANGO-SEC-020",
    name="Django Code Injection via eval()",
    severity="CRITICAL",
    category="django",
    cwe="CWE-95",
    tags="python,django,code-injection,eval,OWASP-A03,CWE-95",
    message="User input flows to eval(). Never use eval() with untrusted data.",
    owasp="A03:2021",
)
def detect_django_eval_injection():
    """Detects Django request data flowing to eval()."""
    return flows(
        from_sources=_DJANGO_SOURCES,
        to_sinks=[
            Builtins.method("eval").tracks(0),
            calls("eval"),
        ],
        sanitized_by=[
            calls("ast.literal_eval"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
