from rules.python_decorators import python_rule
from codepathfinder import calls, QueryType


@python_rule(
    id="PYTHON-LANG-SEC-070",
    name="Socket Bind to All Interfaces",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-200",
    tags="python,socket,bind,network,CWE-200",
    message="Socket bound to 0.0.0.0 (all interfaces). Bind to specific interface in production.",
    owasp="A05:2021",
)
def detect_bind_all():
    """Detects socket.bind to 0.0.0.0 or empty string."""
    return calls("*.bind", match_position={"0[0]": "0.0.0.0"})
