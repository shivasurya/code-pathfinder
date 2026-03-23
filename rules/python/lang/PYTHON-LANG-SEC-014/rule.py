from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class SocketModule(QueryType):
    fqns = ["socket"]


@python_rule(
    id="PYTHON-LANG-SEC-014",
    name="Python Reverse Shell Detected",
    severity="CRITICAL",
    category="lang",
    cwe="CWE-506",
    tags="python,reverse-shell,backdoor,CWE-506",
    message="Reverse shell pattern detected. This may be a backdoor.",
    owasp="A03:2021",
)
def detect_reverse_shell():
    """Detects socket-based reverse shell patterns."""
    return SocketModule.method("socket")
