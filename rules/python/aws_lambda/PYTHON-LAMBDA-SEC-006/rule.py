from codepathfinder.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

# Lambda event sources — event dict is the primary untrusted input
_LAMBDA_SOURCES = [
    calls("event.get"),
    calls("event.items"),
    calls("event.values"),
    calls("event.keys"),
    calls("*.get"),
]


@python_rule(
    id="PYTHON-LAMBDA-SEC-006",
    name="Lambda Command Injection via loop.subprocess_exec",
    severity="HIGH",
    category="aws_lambda",
    cwe="CWE-78",
    tags="python,aws,lambda,command-injection,asyncio-loop,OWASP-A03,CWE-78",
    message="Lambda event data flows to loop.subprocess_exec().",
    owasp="A03:2021",
)
def detect_lambda_loop_subprocess():
    """Detects Lambda event data flowing to event loop subprocess_exec."""
    return flows(
        from_sources=_LAMBDA_SOURCES,
        to_sinks=[
            calls("*.subprocess_exec"),
        ],
        sanitized_by=[
            calls("shlex.quote"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
