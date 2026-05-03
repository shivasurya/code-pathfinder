from codepathfinder.python_decorators import python_rule
from codepathfinder import calls, QueryType

class SSLModule(QueryType):
    fqns = ["ssl"]


@python_rule(
    id="PYTHON-LANG-SEC-051",
    name="Weak SSL/TLS Protocol Version",
    severity="HIGH",
    category="lang",
    cwe="CWE-326",
    tags="python,ssl,weak-tls,protocol-version,CWE-326",
    message="Weak SSL/TLS version detected (SSLv2/3 or TLSv1/1.1). Use TLS 1.2+ minimum.",
    owasp="A02:2021",
)
def detect_weak_ssl():
    """Detects SSLContext with weak protocol versions."""
    return SSLModule.method("SSLContext").where(0, "ssl.PROTOCOL_SSLv2")
