from rules.python_decorators import python_rule
from codepathfinder import calls, QueryType

class SSLModule(QueryType):
    fqns = ["ssl"]


@python_rule(
    id="PYTHON-LANG-SEC-052",
    name="Deprecated ssl.wrap_socket()",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-326",
    tags="python,ssl,wrap-socket,deprecated,cwe-326",
    message="ssl.wrap_socket() is deprecated since Python 3.7. Use SSLContext.wrap_socket() instead.",
    owasp="A02:2021",
)
def detect_wrap_socket():
    """Detects deprecated ssl.wrap_socket() usage."""
    return SSLModule.method("wrap_socket")
