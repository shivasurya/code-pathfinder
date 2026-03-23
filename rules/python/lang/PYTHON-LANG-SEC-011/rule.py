from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class OSModule(QueryType):
    fqns = ["os"]


@python_rule(
    id="PYTHON-LANG-SEC-011",
    name="Dangerous os.exec*() Call",
    severity="HIGH",
    category="lang",
    cwe="CWE-78",
    tags="python,os-exec,command-injection,OWASP-A03,CWE-78",
    message="os.exec*() detected. These replace the current process with a new one.",
    owasp="A03:2021",
)
def detect_os_exec():
    """Detects os.execl/execle/execlp/execlpe/execv/execve/execvp/execvpe calls."""
    return OSModule.method("execl", "execle", "execlp", "execlpe",
                           "execv", "execve", "execvp", "execvpe")
