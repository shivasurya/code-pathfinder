from codepathfinder.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class AsyncioModule(QueryType):
    fqns = ["asyncio"]

# Lambda event sources — event dict is the primary untrusted input
_LAMBDA_SOURCES = [
    calls("event.get"),
    calls("event.items"),
    calls("event.values"),
    calls("event.keys"),
    calls("*.get"),
]


@python_rule(
    id="PYTHON-LAMBDA-SEC-005",
    name="Lambda Command Injection via asyncio exec",
    severity="CRITICAL",
    category="aws_lambda",
    cwe="CWE-78",
    tags="python,aws,lambda,command-injection,asyncio,OWASP-A03,CWE-78",
    message="Lambda event data flows to asyncio.create_subprocess_exec().",
    owasp="A03:2021",
)
def detect_lambda_asyncio_exec():
    """Detects Lambda event data flowing to asyncio exec subprocess."""
    return flows(
        from_sources=_LAMBDA_SOURCES,
        to_sinks=[
            AsyncioModule.method("create_subprocess_exec"),
        ],
        sanitized_by=[
            calls("shlex.quote"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
