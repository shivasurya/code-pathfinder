from rules.python_decorators import python_rule
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
    id="PYTHON-LAMBDA-SEC-003",
    name="Lambda Command Injection via os.spawn*()",
    severity="CRITICAL",
    category="aws_lambda",
    cwe="CWE-78",
    tags="python,aws,lambda,command-injection,os-spawn,OWASP-A03,CWE-78",
    message="Lambda event data flows to os.spawn*(). Use subprocess with list args instead.",
    owasp="A03:2021",
)
def detect_lambda_os_spawn():
    """Detects Lambda event data flowing to os.spawn*() functions."""
    return flows(
        from_sources=_LAMBDA_SOURCES,
        to_sinks=[
            OSModule.method("spawnl", "spawnle", "spawnlp", "spawnlpe",
                            "spawnv", "spawnve", "spawnvp", "spawnvpe"),
        ],
        sanitized_by=[
            calls("shlex.quote"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
