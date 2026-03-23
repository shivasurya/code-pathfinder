from rules.python_decorators import python_rule
from codepathfinder import calls, Or, QueryType

class FlaskCORS(QueryType):
    fqns = ["flask_cors"]


@python_rule(
    id="PYTHON-FLASK-AUDIT-004",
    name="Flask CORS Wildcard Origin",
    severity="MEDIUM",
    category="flask",
    cwe="CWE-942",
    tags="python,flask,cors,wildcard,CWE-942",
    message="CORS configured with wildcard origin '*'. Restrict to specific domains.",
    owasp="A05:2021",
)
def detect_flask_cors_wildcard():
    """Detects CORS(app, origins='*')."""
    return FlaskCORS.method("CORS").where("origins", "*")
