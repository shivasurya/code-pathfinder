from rules.python_decorators import python_rule
from codepathfinder import calls, Or, QueryType


@python_rule(
    id="PYTHON-FLASK-XSS-001",
    name="Flask Direct Use of Jinja2",
    severity="MEDIUM",
    category="flask",
    cwe="CWE-79",
    tags="python,flask,jinja2,xss,audit,cwe-79",
    message="Direct Jinja2 Environment usage bypasses Flask's auto-escaping. Use Flask's render_template().",
    owasp="A07:2021",
)
def detect_flask_direct_jinja2():
    """Detects direct Jinja2 Environment creation."""
    return Or(
        calls("Environment"),
        calls("jinja2.Environment"),
    )
