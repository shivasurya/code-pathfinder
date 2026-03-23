from rules.python_decorators import python_rule
from codepathfinder import calls, QueryType

class ShelveModule(QueryType):
    fqns = ["shelve"]


@python_rule(
    id="PYTHON-LANG-SEC-045",
    name="shelve Usage Detected",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-502",
    tags="python,shelve,deserialization,pickle,CWE-502",
    message="shelve.open() uses pickle internally. Not safe for untrusted data.",
    owasp="A08:2021",
)
def detect_shelve():
    """Detects shelve.open() which uses pickle internally."""
    return ShelveModule.method("open")
