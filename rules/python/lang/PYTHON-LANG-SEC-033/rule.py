from codepathfinder.python_decorators import python_rule
from codepathfinder import calls, QueryType

class HashlibModule(QueryType):
    fqns = ["hashlib"]


@python_rule(
    id="PYTHON-LANG-SEC-033",
    name="SHA-224 Weak Hash",
    severity="LOW",
    category="lang",
    cwe="CWE-327",
    tags="python,sha224,weak-hash,CWE-327",
    message="SHA-224 provides only 112-bit security. Consider SHA-256 or SHA-3.",
    owasp="A02:2021",
)
def detect_sha224():
    """Detects hashlib.sha224() and sha3_224() usage."""
    return HashlibModule.method("sha224", "sha3_224")
