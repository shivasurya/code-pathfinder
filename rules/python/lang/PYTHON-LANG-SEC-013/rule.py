from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class OSModule(QueryType):
    fqns = ["os"]


@python_rule(
    id="PYTHON-LANG-SEC-013",
    name="System Call with Wildcard",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-78",
    tags="python,wildcard,command-injection,CWE-78",
    message="os.system() with wildcard (*) detected. Wildcards may lead to unintended file inclusion.",
    owasp="A03:2021",
)
def detect_system_wildcard():
    """Detects os.system() calls (audit for wildcard usage)."""
    return OSModule.method("system")
