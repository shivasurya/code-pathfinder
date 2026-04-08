from rules.python_decorators import python_rule
from codepathfinder import calls, QueryType


@python_rule(
    id="PYTHON-LANG-SEC-113",
    name="Host Header Used for Access Control",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-287",
    tags="python,host-header,authentication,origin-validation,CWE-287,OWASP-A07",
    message="HTTP_HOST header used for access control or routing. Host headers are attacker-controlled and must not be trusted for security decisions.",
    owasp="A07:2021",
)
def detect_host_header_auth():
    """Detects use of HTTP_HOST header via request.environ.get for access control."""
    return calls("*.environ.get", "environ.get", match_position={"0": "HTTP_HOST"})
