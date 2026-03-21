from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class FlaskModule(QueryType):
    fqns = ["flask"]


@python_rule(
    id="PYTHON-FLASK-SEC-012",
    name="Flask Open Redirect",
    severity="MEDIUM",
    category="flask",
    cwe="CWE-601",
    tags="python,flask,open-redirect,owasp-a01,cwe-601",
    message="User input flows to redirect(). Validate redirect URLs against an allowlist.",
    owasp="A01:2021",
)
def detect_flask_open_redirect():
    """Detects Flask request data flowing to redirect()."""
    return flows(
        from_sources=[
            calls("request.args.get"),
            calls("request.form.get"),
            calls("request.values.get"),
            calls("request.get_json"),
        ],
        to_sinks=[
            FlaskModule.method("redirect").tracks(0),
            calls("redirect"),
        ],
        sanitized_by=[
            calls("url_for"),
            calls("url_has_allowed_host_and_scheme"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
