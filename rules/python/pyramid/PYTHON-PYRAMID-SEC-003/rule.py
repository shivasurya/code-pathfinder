from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

_PYRAMID_SOURCES = [
    calls("request.params.get"),
    calls("request.params"),
    calls("request.GET.get"),
    calls("request.POST.get"),
    calls("request.matchdict.get"),
    calls("request.json_body.get"),
    calls("*.params.get"),
    calls("*.params"),
]


@python_rule(
    id="PYTHON-PYRAMID-SEC-003",
    name="Pyramid SQLAlchemy SQL Injection",
    severity="CRITICAL",
    category="pyramid",
    cwe="CWE-89",
    tags="python,pyramid,sqlalchemy,sql-injection,owasp-a03,cwe-89",
    message="User input flows to raw SQL in SQLAlchemy. Use parameterized queries with bindparams().",
    owasp="A03:2021",
)
def detect_pyramid_sqli():
    """Detects request data flowing to SQLAlchemy raw SQL."""
    return flows(
        from_sources=_PYRAMID_SOURCES,
        to_sinks=[
            calls("*.filter"),
            calls("*.order_by"),
            calls("*.group_by"),
            calls("*.having"),
            calls("*.execute"),
        ],
        sanitized_by=[
            calls("bindparams"),
            calls("*.bindparams"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="local",
    )
