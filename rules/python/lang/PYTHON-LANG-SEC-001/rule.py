from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class Builtins(QueryType):
    fqns = ["builtins"]


@python_rule(
    id="PYTHON-LANG-SEC-001",
    name="Dangerous eval() Detected",
    severity="HIGH",
    category="lang",
    cwe="CWE-95",
    tags="python,eval,code-injection,OWASP-A03,CWE-95",
    message="eval() detected. Avoid eval() on untrusted input. Use ast.literal_eval() for safe parsing.",
    owasp="A03:2021",
)
def detect_eval():
    """Detects eval() calls."""
    return Builtins.method("eval")
