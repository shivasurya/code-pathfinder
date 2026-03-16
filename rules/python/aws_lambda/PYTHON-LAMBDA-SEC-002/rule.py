from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class SubprocessModule(QueryType):
    fqns = ["subprocess"]

# Lambda event sources — event dict is the primary untrusted input
_LAMBDA_SOURCES = [
    calls("event.get"),
    calls("event.items"),
    calls("event.values"),
    calls("event.keys"),
    calls("*.get"),
]


@python_rule(
    id="PYTHON-LAMBDA-SEC-002",
    name="Lambda Command Injection via subprocess",
    severity="CRITICAL",
    category="aws_lambda",
    cwe="CWE-78",
    tags="python,aws,lambda,command-injection,subprocess,owasp-a03,cwe-78",
    message="Lambda event data flows to subprocess call. Use shlex.quote() or list args.",
    owasp="A03:2021",
)
def detect_lambda_subprocess():
    """Detects Lambda event data flowing to subprocess functions."""
    return flows(
        from_sources=_LAMBDA_SOURCES,
        to_sinks=[
            SubprocessModule.method("call", "check_call", "check_output",
                                    "run", "Popen", "getoutput", "getstatusoutput"),
        ],
        sanitized_by=[
            calls("shlex.quote"),
            calls("shlex.split"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
