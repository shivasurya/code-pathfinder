from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

_LAMBDA_SOURCES = [
    calls("event.get"),
    calls("event.items"),
    calls("event.values"),
    calls("*.get"),
]


@python_rule(
    id="PYTHON-LAMBDA-SEC-016",
    name="Lambda DynamoDB Filter Injection",
    severity="HIGH",
    category="aws_lambda",
    cwe="CWE-943",
    tags="python,aws,lambda,dynamodb,nosql-injection,OWASP-A03,CWE-943",
    message="Lambda event data flows to DynamoDB scan/query filter. Validate input.",
    owasp="A03:2021",
)
def detect_lambda_dynamodb_injection():
    """Detects Lambda event data flowing to DynamoDB scan/query filters."""
    return flows(
        from_sources=_LAMBDA_SOURCES,
        to_sinks=[
            calls("*.scan"),
            calls("*.query"),
            calls("table.scan"),
            calls("table.query"),
        ],
        sanitized_by=[],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
