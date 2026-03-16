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
    id="PYTHON-LAMBDA-SEC-015",
    name="Lambda Tainted SQL String Construction",
    severity="HIGH",
    category="aws_lambda",
    cwe="CWE-89",
    tags="python,aws,lambda,sql-injection,string-format,owasp-a03,cwe-89",
    message="Lambda event data used in SQL string construction. Use parameterized queries.",
    owasp="A03:2021",
)
def detect_lambda_tainted_sql():
    """Detects Lambda event data in string formatting that reaches execute()."""
    return flows(
        from_sources=_LAMBDA_SOURCES,
        to_sinks=[
            calls("cursor.execute"),
            calls("*.execute"),
        ],
        sanitized_by=[
            calls("escape"),
            calls("int"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
