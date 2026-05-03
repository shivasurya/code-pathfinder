from codepathfinder.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets


@python_rule(
    id="PYTHON-LANG-SEC-004",
    name="Dangerous globals() Usage",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-96",
    tags="python,globals,code-injection,CWE-96",
    message="globals() passed to a function. This may allow arbitrary attribute access.",
    owasp="A03:2021",
)
def detect_globals():
    """Detects globals() being passed to functions."""
    return calls("globals")
