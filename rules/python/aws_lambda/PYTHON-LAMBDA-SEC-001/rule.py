from codepathfinder.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class OSModule(QueryType):
    fqns = ["os"]

# Lambda event sources — event dict is the primary untrusted input
_LAMBDA_SOURCES = [
    calls("event.get"),
    calls("event.items"),
    calls("event.values"),
    calls("event.keys"),
    calls("*.get"),
]


@python_rule(
    id="PYTHON-LAMBDA-SEC-001",
    name="Lambda Command Injection via os.system()",
    severity="CRITICAL",
    category="aws_lambda",
    cwe="CWE-78",
    tags="python,aws,lambda,command-injection,os-system,OWASP-A03,CWE-78",
    message="Lambda event data flows to os.system(). Use subprocess with list args instead.",
    owasp="A03:2021",
)
def detect_lambda_os_system():
    """Detects Lambda event data flowing to os.system()/os.popen()."""
    return flows(
        from_sources=_LAMBDA_SOURCES,
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
