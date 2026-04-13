from codepathfinder.python_decorators import python_rule
from codepathfinder import calls, QueryType

class HashlibModule(QueryType):
    fqns = ["hashlib"]


@python_rule(
    id="PYTHON-LANG-SEC-030",
    name="Insecure MD5 Hash Usage",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-327",
    tags="python,md5,weak-hash,cryptography,OWASP-A02,CWE-327",
    message="MD5 is cryptographically broken. Use SHA-256 or SHA-3 for security-sensitive hashing.",
    owasp="A02:2021",
)
def detect_md5():
    """Detects hashlib.md5() usage."""
    return HashlibModule.method("md5")
