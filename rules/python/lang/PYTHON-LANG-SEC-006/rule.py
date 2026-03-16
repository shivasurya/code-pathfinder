from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets


@python_rule(
    id="PYTHON-LANG-SEC-006",
    name="Dangerous Annotations Usage",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-95",
    tags="python,annotations,pep-563,cwe-95",
    message="typing.get_type_hints() may execute string annotations. Be cautious with untrusted code.",
    owasp="A03:2021",
)
def detect_dangerous_annotations():
    """Detects typing.get_type_hints() which evaluates string annotations."""
    return calls("typing.get_type_hints", "get_type_hints")
