from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class PickleModule(QueryType):
    fqns = ["pickle", "_pickle", "cPickle"]

_LAMBDA_SOURCES = [
    calls("event.get"),
    calls("event.items"),
    calls("event.values"),
    calls("*.get"),
]


@python_rule(
    id="PYTHON-LAMBDA-SEC-023",
    name="Lambda Pickle Deserialization",
    severity="CRITICAL",
    category="aws_lambda",
    cwe="CWE-502",
    tags="python,aws,lambda,deserialization,pickle,OWASP-A08,CWE-502",
    message="Lambda event data flows to pickle deserialization. Use JSON instead.",
    owasp="A08:2021",
)
def detect_lambda_pickle():
    """Detects Lambda event data flowing to pickle deserialization."""
    return flows(
        from_sources=_LAMBDA_SOURCES,
        to_sinks=[
            PickleModule.method("loads", "load"),
        ],
        sanitized_by=[],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
