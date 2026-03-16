from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets


@python_rule(
    id="PYTHON-LANG-SEC-023",
    name="Dangerous subinterpreters run_string",
    severity="HIGH",
    category="lang",
    cwe="CWE-95",
    tags="python,subinterpreters,code-execution,cwe-95",
    message="subinterpreters.run_string() detected. Avoid executing untrusted code strings.",
    owasp="A03:2021",
)
def detect_subinterpreters():
    """Detects _xxsubinterpreters.run_string usage."""
    return calls("_xxsubinterpreters.run_string", "subinterpreters.run_string")
