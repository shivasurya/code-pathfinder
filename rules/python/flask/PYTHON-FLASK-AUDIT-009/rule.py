from rules.python_decorators import python_rule
from codepathfinder import calls, Or, QueryType


@python_rule(
    id="PYTHON-FLASK-AUDIT-009",
    name="Flask Cookie Without Secure Flags",
    severity="MEDIUM",
    category="flask",
    cwe="CWE-614",
    tags="python,flask,cookie,secure,httponly,cwe-614",
    message="Cookie set without secure=True or httponly=True. Set both flags for session cookies.",
    owasp="A05:2021",
)
def detect_flask_insecure_cookie():
    """Detects set_cookie() with secure=False or httponly=False."""
    return Or(
        calls("*.set_cookie", match_name={"secure": False}),
        calls("*.set_cookie", match_name={"httponly": False}),
    )
