from rules.python_decorators import python_rule
from codepathfinder import calls, QueryType

class SSLModule(QueryType):
    fqns = ["ssl"]


@python_rule(
    id="PYTHON-LANG-SEC-050",
    name="Unverified SSL Context",
    severity="HIGH",
    category="lang",
    cwe="CWE-295",
    tags="python,ssl,unverified-context,certificate,owasp-a07,cwe-295",
    message="ssl._create_unverified_context() disables certificate verification. Use ssl.create_default_context().",
    owasp="A07:2021",
)
def detect_unverified_ssl():
    """Detects ssl._create_unverified_context() usage."""
    return SSLModule.method("_create_unverified_context")
