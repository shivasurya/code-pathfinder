from codepathfinder.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class SubprocessModule(QueryType):
    fqns = ["subprocess"]


@python_rule(
    id="PYTHON-LANG-SEC-020",
    name="Dangerous subprocess Usage",
    severity="HIGH",
    category="lang",
    cwe="CWE-78",
    tags="python,subprocess,command-injection,OWASP-A03,CWE-78",
    message="subprocess call detected. Ensure arguments are not user-controlled.",
    owasp="A03:2021",
)
def detect_subprocess():
    """Detects subprocess module calls."""
    return SubprocessModule.method("call", "check_call", "check_output",
                                   "run", "Popen", "getoutput", "getstatusoutput")
