from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class OSModule(QueryType):
    fqns = ["os"]


@python_rule(
    id="PYTHON-LANG-SEC-010",
    name="Dangerous os.system() Call",
    severity="HIGH",
    category="lang",
    cwe="CWE-78",
    tags="python,os-system,command-injection,owasp-a03,cwe-78",
    message="os.system() detected. Use subprocess.run() with list arguments instead.",
    owasp="A03:2021",
)
def detect_os_system():
    """Detects os.system() and os.popen() calls."""
    return OSModule.method("system", "popen", "popen2", "popen3", "popen4")
