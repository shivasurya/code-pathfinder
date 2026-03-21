from rules.python_decorators import python_rule
from codepathfinder import calls, Or, QueryType


@python_rule(
    id="PYTHON-FLASK-XSS-002",
    name="Flask Explicit Unescape with Markup",
    severity="MEDIUM",
    category="flask",
    cwe="CWE-79",
    tags="python,flask,markup,xss,audit,cwe-79",
    message="Markup() bypasses auto-escaping. Ensure input is trusted before wrapping in Markup().",
    owasp="A07:2021",
)
def detect_flask_markup_usage():
    """Detects Markup() usage which bypasses escaping."""
    return Or(
        calls("Markup"),
        calls("markupsafe.Markup"),
    )
