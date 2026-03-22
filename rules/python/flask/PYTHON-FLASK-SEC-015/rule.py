from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class FlaskModule(QueryType):
    fqns = ["flask"]


@python_rule(
    id="PYTHON-FLASK-SEC-015",
    name="Flask Unsanitized Input in Response",
    severity="MEDIUM",
    category="flask",
    cwe="CWE-79",
    tags="python,flask,xss,unsanitized-input,OWASP-A07,CWE-79",
    message="User input returned directly in response without escaping. Use markupsafe.escape().",
    owasp="A07:2021",
)
def detect_flask_unsanitized_response():
    """Detects Flask request data returned directly in response."""
    return flows(
        from_sources=[
            calls("request.args.get"),
            calls("request.form.get"),
            calls("request.values.get"),
        ],
        to_sinks=[
            FlaskModule.method("make_response", "jsonify").tracks(0),
            calls("make_response"),
            calls("jsonify"),
        ],
        sanitized_by=[
            calls("escape"),
            calls("markupsafe.escape"),
            calls("html.escape"),
            calls("bleach.clean"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
