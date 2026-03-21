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
    id="PYTHON-LAMBDA-SEC-020",
    name="Lambda Tainted HTML Response",
    severity="MEDIUM",
    category="aws_lambda",
    cwe="CWE-79",
    tags="python,aws,lambda,xss,html,owasp-a03,cwe-79",
    message="Lambda event data in HTML response body. Sanitize output with html.escape().",
    owasp="A03:2021",
)
def detect_lambda_html_response():
    """Detects Lambda event data in HTML response returned to API Gateway."""
    return flows(
        from_sources=_LAMBDA_SOURCES,
        to_sinks=[
            calls("json.dumps"),
        ],
        sanitized_by=[
            calls("html.escape"),
            calls("escape"),
            calls("markupsafe.escape"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
