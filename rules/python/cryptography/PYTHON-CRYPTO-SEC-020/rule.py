from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class CryptoRSA(QueryType):
    fqns = ["cryptography.hazmat.primitives.asymmetric.rsa"]


@python_rule(
    id="PYTHON-CRYPTO-SEC-020",
    name="RSA Key Generation (Audit)",
    severity="MEDIUM",
    category="cryptography",
    cwe="CWE-326",
    tags="python,cryptography,rsa,key-size,audit,cwe-326",
    message="RSA key generation detected. Ensure key_size >= 2048 bits.",
    owasp="A02:2021",
)
def detect_rsa_keygen_crypto():
    """Audit: detects RSA key generation in cryptography lib."""
    return CryptoRSA.method("generate_private_key")
