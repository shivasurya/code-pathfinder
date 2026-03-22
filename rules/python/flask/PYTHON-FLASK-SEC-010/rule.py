from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class Builtins(QueryType):
    fqns = ["builtins"]


@python_rule(
    id="PYTHON-FLASK-SEC-010",
    name="Flask NaN Injection",
    severity="LOW",
    category="flask",
    cwe="CWE-704",
    tags="python,flask,nan-injection,type-confusion,CWE-704",
    message="User input flows to float() which may produce NaN/Inf. Validate numeric input.",
    owasp="A03:2021",
)
def detect_flask_nan_injection():
    """Detects Flask request data flowing to float() conversion."""
    return flows(
        from_sources=[
            calls("request.args.get"),
            calls("request.form.get"),
            calls("request.values.get"),
        ],
        to_sinks=[
            Builtins.method("float").tracks(0),
            calls("float"),
        ],
        sanitized_by=[
            calls("int"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
