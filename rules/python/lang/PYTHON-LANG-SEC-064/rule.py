from rules.python_decorators import python_rule
from codepathfinder import calls, QueryType

class TelnetModule(QueryType):
    fqns = ["telnetlib"]


@python_rule(
    id="PYTHON-LANG-SEC-064",
    name="telnetlib Usage Detected",
    severity="HIGH",
    category="lang",
    cwe="CWE-319",
    tags="python,telnet,insecure-transport,plaintext,CWE-319",
    message="telnetlib transmits data in plaintext. Use SSH (paramiko) instead.",
    owasp="A02:2021",
)
def detect_telnet():
    """Detects telnetlib.Telnet usage."""
    return TelnetModule.method("Telnet")
