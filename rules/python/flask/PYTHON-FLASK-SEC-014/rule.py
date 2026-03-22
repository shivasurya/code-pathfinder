from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class FlaskModule(QueryType):
    fqns = ["flask"]


@python_rule(
    id="PYTHON-FLASK-SEC-014",
    name="Flask Server-Side Template Injection",
    severity="CRITICAL",
    category="flask",
    cwe="CWE-1336",
    tags="python,flask,ssti,template-injection,rce,OWASP-A03,CWE-1336",
    message="User input flows to render_template_string(). Use render_template() with separate template files.",
    owasp="A03:2021",
)
def detect_flask_ssti():
    """Detects Flask request data flowing to render_template_string()."""
    return flows(
        from_sources=[
            calls("request.args.get"),
            calls("request.form.get"),
            calls("request.values.get"),
            calls("request.get_json"),
        ],
        to_sinks=[
            FlaskModule.method("render_template_string").tracks(0),
            calls("render_template_string"),
        ],
        sanitized_by=[],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
