from codepathfinder.python_decorators import python_rule
from codepathfinder import calls, QueryType

class UUIDModule(QueryType):
    fqns = ["uuid"]


@python_rule(
    id="PYTHON-LANG-SEC-100",
    name="Insecure UUID Version (uuid1)",
    severity="LOW",
    category="lang",
    cwe="CWE-200",
    tags="python,uuid,mac-address,insufficiently-random,CWE-200",
    message="uuid.uuid1() leaks the host MAC address and uses predictable timestamps. Use uuid.uuid4() for random UUIDs.",
    owasp="A05:2021",
)
def detect_uuid1():
    """Detects uuid.uuid1() which leaks MAC address."""
    return UUIDModule.method("uuid1")
