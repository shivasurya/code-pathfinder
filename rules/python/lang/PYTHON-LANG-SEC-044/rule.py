from rules.python_decorators import python_rule
from codepathfinder import calls, QueryType

class MarshalModule(QueryType):
    fqns = ["marshal"]


@python_rule(
    id="PYTHON-LANG-SEC-044",
    name="marshal Usage Detected",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-502",
    tags="python,marshal,deserialization,CWE-502",
    message="marshal.loads/load detected. Marshal is not secure against erroneous or malicious data.",
    owasp="A08:2021",
)
def detect_marshal():
    """Detects marshal.loads/load/dump/dumps usage."""
    return MarshalModule.method("loads", "load")
