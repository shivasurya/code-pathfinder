from codepathfinder.python_decorators import python_rule
from codepathfinder import calls, QueryType

class HashlibModule(QueryType):
    fqns = ["hashlib"]


@python_rule(
    id="PYTHON-LANG-SEC-032",
    name="Insecure Hash via hashlib.new()",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-327",
    tags="python,weak-hash,hashlib-new,CWE-327",
    message="hashlib.new() with insecure algorithm. Use SHA-256 or SHA-3.",
    owasp="A02:2021",
)
def detect_hashlib_new_insecure():
    """Detects hashlib.new() which may use insecure algorithms."""
    return HashlibModule.method("new")
