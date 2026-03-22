from rules.python_decorators import python_rule
from codepathfinder import calls, QueryType


@python_rule(
    id="PYTHON-LANG-SEC-073",
    name="multiprocessing Connection.recv()",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-502",
    tags="python,multiprocessing,recv,deserialization,CWE-502",
    message="Connection.recv() uses pickle internally. Not safe for untrusted connections.",
    owasp="A08:2021",
)
def detect_conn_recv():
    """Detects multiprocessing Connection.recv() which uses pickle."""
    return calls("*.recv", "multiprocessing.connection.Connection.recv")
