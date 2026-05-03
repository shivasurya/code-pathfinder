from codepathfinder.python_decorators import python_rule
from codepathfinder import calls, Or, QueryType


@python_rule(
    id="PYTHON-FLASK-AUDIT-008",
    name="Flask render_template_string Usage",
    severity="MEDIUM",
    category="flask",
    cwe="CWE-1336",
    tags="python,flask,template,ssti,audit,CWE-1336",
    message="render_template_string() detected. Prefer render_template() with separate template files.",
    owasp="A03:2021",
)
def detect_flask_render_template_string():
    """Audit: Detects any usage of render_template_string()."""
    return Or(
        calls("render_template_string"),
        calls("flask.render_template_string"),
    )
