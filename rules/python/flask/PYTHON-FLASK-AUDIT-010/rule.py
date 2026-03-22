from rules.python_decorators import python_rule
from codepathfinder import calls, Or, QueryType


@python_rule(
    id="PYTHON-FLASK-AUDIT-010",
    name="Flask WTF CSRF Disabled",
    severity="HIGH",
    category="flask",
    cwe="CWE-352",
    tags="python,flask,csrf,wtf,CWE-352",
    message="WTF_CSRF_ENABLED set to False. CSRF protection should always be enabled.",
    owasp="A05:2021",
)
def detect_flask_wtf_csrf_disabled():
    """Detects WTF_CSRF_ENABLED = False. Pattern match on config assignment."""
    return calls("*.config.__setitem__", match_position={0: "WTF_CSRF_ENABLED"})
