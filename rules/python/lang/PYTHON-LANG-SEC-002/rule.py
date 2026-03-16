from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class Builtins(QueryType):
    fqns = ["builtins"]


@python_rule(
    id="PYTHON-LANG-SEC-002",
    name="Dangerous exec() Detected",
    severity="HIGH",
    category="lang",
    cwe="CWE-95",
    tags="python,exec,code-injection,owasp-a03,cwe-95",
    message="exec() detected. Never execute dynamically constructed code.",
    owasp="A03:2021",
)
def detect_exec():
    """Detects exec() calls."""
    return Builtins.method("exec")
