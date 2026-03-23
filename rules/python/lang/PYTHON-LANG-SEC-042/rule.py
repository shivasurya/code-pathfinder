from rules.python_decorators import python_rule
from codepathfinder import calls, QueryType

class PickleModule(QueryType):
    fqns = ["pickle", "_pickle", "cPickle"]

class JsonPickleModule(QueryType):
    fqns = ["jsonpickle"]


@python_rule(
    id="PYTHON-LANG-SEC-042",
    name="jsonpickle Usage Detected",
    severity="HIGH",
    category="lang",
    cwe="CWE-502",
    tags="python,jsonpickle,deserialization,rce,CWE-502",
    message="jsonpickle.decode() detected. jsonpickle can execute arbitrary code. Use json instead.",
    owasp="A08:2021",
)
def detect_jsonpickle():
    """Detects jsonpickle.decode/loads usage."""
    return JsonPickleModule.method("decode", "loads")
