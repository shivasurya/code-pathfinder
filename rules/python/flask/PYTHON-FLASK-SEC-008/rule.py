from codepathfinder.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class FlaskModule(QueryType):
    fqns = ["flask"]


@python_rule(
    id="PYTHON-FLASK-SEC-008",
    name="Flask XSS via Raw HTML Concatenation",
    severity="HIGH",
    category="flask",
    cwe="CWE-79",
    tags="python,flask,xss,html,OWASP-A07,CWE-79",
    message="User input concatenated into HTML response. Use render_template() or markupsafe.escape().",
    owasp="A07:2021",
)
def detect_flask_xss_html_concat():
    """Detects Flask request data concatenated into HTML and returned in response."""
    return flows(
        from_sources=[
            calls("request.args.get"),
            calls("request.form.get"),
            calls("request.values.get"),
            calls("request.get_json"),
        ],
        to_sinks=[
            FlaskModule.method("make_response").tracks(0),
            calls("make_response"),
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
