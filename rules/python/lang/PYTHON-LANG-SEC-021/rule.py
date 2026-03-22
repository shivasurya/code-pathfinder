from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class SubprocessModule(QueryType):
    fqns = ["subprocess"]


@python_rule(
    id="PYTHON-LANG-SEC-021",
    name="subprocess with shell=True",
    severity="HIGH",
    category="lang",
    cwe="CWE-78",
    tags="python,subprocess,shell-true,command-injection,OWASP-A03,CWE-78",
    message="subprocess called with shell=True. This is vulnerable to shell injection.",
    owasp="A03:2021",
)
def detect_subprocess_shell_true():
    """Detects subprocess calls with shell=True."""
    return SubprocessModule.method("call", "check_call", "check_output",
                                   "run", "Popen").where("shell", True)
