from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class CryptoEC(QueryType):
    fqns = ["cryptography.hazmat.primitives.asymmetric.ec"]


@python_rule(
    id="PYTHON-CRYPTO-SEC-022",
    name="EC Key Generation (Audit)",
    severity="LOW",
    category="cryptography",
    cwe="CWE-326",
    tags="python,cryptography,ec,elliptic-curve,key-size,audit,CWE-326",
    message="EC key generation detected. Ensure curve >= 224 bits (avoid SECP192R1).",
    owasp="A02:2021",
)
def detect_ec_keygen_crypto():
    """Audit: detects EC key generation in cryptography lib."""
    return CryptoEC.method("generate_private_key")
