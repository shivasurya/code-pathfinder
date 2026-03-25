from rules.python_decorators import python_rule
from codepathfinder import calls, QueryType


@python_rule(
    id="PYTHON-LANG-SEC-133",
    name="RSA PKCS1v15 Deprecated Padding Detected",
    severity="HIGH",
    category="lang",
    cwe="CWE-327",
    tags="python,cryptography,rsa,pkcs1v15,padding,bleichenbacher,CWE-327,OWASP-A02",
    message="PKCS1v15 padding detected for RSA encryption or signing. PKCS#1 v1.5 is vulnerable to Bleichenbacher's attack. Use OAEP for encryption or PSS for signatures.",
    owasp="A02:2021",
)
def detect_pkcs1v15():
    """Detects use of deprecated PKCS1v15 padding for RSA operations."""
    return calls("*.PKCS1v15", "padding.PKCS1v15", "PKCS1v15")
