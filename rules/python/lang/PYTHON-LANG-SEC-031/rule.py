from rules.python_decorators import python_rule
from codepathfinder import calls, QueryType

class HashlibModule(QueryType):
    fqns = ["hashlib"]


@python_rule(
    id="PYTHON-LANG-SEC-031",
    name="Insecure SHA1 Hash Usage",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-327",
    tags="python,sha1,weak-hash,cryptography,owasp-a02,cwe-327",
    message="SHA-1 is cryptographically weak. Use SHA-256 or SHA-3 instead.",
    owasp="A02:2021",
)
def detect_sha1():
    """Detects hashlib.sha1() usage."""
    return HashlibModule.method("sha1")
