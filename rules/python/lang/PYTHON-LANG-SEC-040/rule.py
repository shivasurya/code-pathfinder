from codepathfinder.python_decorators import python_rule
from codepathfinder import calls, QueryType

class PickleModule(QueryType):
    fqns = ["pickle", "_pickle", "cPickle"]


@python_rule(
    id="PYTHON-LANG-SEC-040",
    name="Pickle Deserialization Detected",
    severity="HIGH",
    category="lang",
    cwe="CWE-502",
    tags="python,pickle,deserialization,rce,OWASP-A08,CWE-502",
    message="pickle.loads/load detected. Pickle can execute arbitrary code. Use json or msgpack instead.",
    owasp="A08:2021",
)
def detect_pickle():
    """Detects pickle.loads/load/Unpickler usage."""
    return PickleModule.method("loads", "load", "Unpickler")
