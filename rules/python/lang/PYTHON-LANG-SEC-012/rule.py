from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class OSModule(QueryType):
    fqns = ["os"]


@python_rule(
    id="PYTHON-LANG-SEC-012",
    name="Dangerous os.spawn*() Call",
    severity="HIGH",
    category="lang",
    cwe="CWE-78",
    tags="python,os-spawn,process-spawn,owasp-a03,cwe-78",
    message="os.spawn*() detected. Use subprocess module instead.",
    owasp="A03:2021",
)
def detect_os_spawn():
    """Detects os.spawnl/spawnle/spawnlp/spawnlpe/spawnv/spawnve/spawnvp/spawnvpe calls."""
    return OSModule.method("spawnl", "spawnle", "spawnlp", "spawnlpe",
                           "spawnv", "spawnve", "spawnvp", "spawnvpe")
