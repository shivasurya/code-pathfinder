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
    id="PYTHON-LAMBDA-SEC-021",
    name="Lambda Tainted HTML String",
    severity="MEDIUM",
    category="aws_lambda",
    cwe="CWE-79",
    tags="python,aws,lambda,xss,html-string,OWASP-A03,CWE-79",
    message="Lambda event data in HTML string construction. Use html.escape().",
    owasp="A03:2021",
)
def detect_lambda_html_string():
    """Detects Lambda event data used to build HTML strings."""
    return flows(
        from_sources=_LAMBDA_SOURCES,
        to_sinks=[
            calls("json.dumps"),
        ],
        sanitized_by=[
            calls("html.escape"),
            calls("escape"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
