from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class CryptoDSA(QueryType):
    fqns = ["cryptography.hazmat.primitives.asymmetric.dsa"]


@python_rule(
    id="PYTHON-CRYPTO-SEC-021",
    name="DSA Key Generation (Audit)",
    severity="MEDIUM",
    category="cryptography",
    cwe="CWE-326",
    tags="python,cryptography,dsa,key-size,audit,cwe-326",
    message="DSA key generation detected. Ensure key_size >= 2048 bits.",
    owasp="A02:2021",
)
def detect_dsa_keygen_crypto():
    """Audit: detects DSA key generation in cryptography lib."""
    return CryptoDSA.method("generate_private_key")
