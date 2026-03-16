from rules.python_decorators import python_rule
from codepathfinder import calls, QueryType

class DillModule(QueryType):
    fqns = ["dill"]


@python_rule(
    id="PYTHON-LANG-SEC-046",
    name="dill Deserialization Detected",
    severity="HIGH",
    category="lang",
    cwe="CWE-502",
    tags="python,dill,deserialization,rce,cwe-502",
    message="dill.loads/load detected. dill extends pickle and can execute arbitrary code.",
    owasp="A08:2021",
)
def detect_dill():
    """Detects dill.loads/load usage."""
    return DillModule.method("loads", "load")
