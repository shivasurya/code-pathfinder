from rules.python_decorators import python_rule
from codepathfinder import calls, Or, QueryType


@python_rule(
    id="PYTHON-FLASK-AUDIT-005",
    name="Flask url_for with _external=True",
    severity="LOW",
    category="flask",
    cwe="CWE-601",
    tags="python,flask,url-for,external,CWE-601",
    message="url_for() with _external=True may expose internal URL schemes. Verify usage.",
    owasp="A01:2021",
)
def detect_flask_url_for_external():
    """Detects url_for(_external=True)."""
    return calls("url_for", match_name={"_external": True})
