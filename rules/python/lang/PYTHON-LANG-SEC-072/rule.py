from codepathfinder.python_decorators import python_rule
from codepathfinder import calls, QueryType


@python_rule(
    id="PYTHON-LANG-SEC-072",
    name="Paramiko exec_command",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-78",
    tags="python,paramiko,ssh,command-execution,CWE-78",
    message="paramiko exec_command() detected. Ensure command is not user-controlled.",
    owasp="A03:2021",
)
def detect_paramiko_exec():
    """Detects paramiko SSHClient.exec_command usage."""
    return calls("*.exec_command")
