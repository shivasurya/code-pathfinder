from rules.python_decorators import python_rule
from codepathfinder import calls, Or, QueryType

class FlaskApp(QueryType):
    fqns = ["flask"]


@python_rule(
    id="PYTHON-FLASK-AUDIT-003",
    name="Flask Bound to All Interfaces",
    severity="MEDIUM",
    category="flask",
    cwe="CWE-200",
    tags="python,flask,network,binding,cwe-200",
    message="Flask app bound to 0.0.0.0 (all interfaces). Bind to 127.0.0.1 in production.",
    owasp="A05:2021",
)
def detect_flask_bind_all():
    """Detects app.run(host='0.0.0.0')."""
    return FlaskApp.method("run").where("host", "0.0.0.0")
