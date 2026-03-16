from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class CryptoRSA(QueryType):
    fqns = ["cryptography.hazmat.primitives.asymmetric.rsa"]

class PyCryptoRSA(QueryType):
    fqns = ["Crypto.PublicKey.RSA", "Cryptodome.PublicKey.RSA"]


@python_rule(
    id="PYTHON-CRYPTO-SEC-023",
    name="RSA Key Generation (PyCryptodome, Audit)",
    severity="MEDIUM",
    category="cryptography",
    cwe="CWE-326",
    tags="python,pycryptodome,rsa,key-size,audit,cwe-326",
    message="RSA key generation detected. Ensure bits >= 3072.",
    owasp="A02:2021",
)
def detect_rsa_keygen_pycrypto():
    """Audit: detects RSA key generation in PyCryptodome."""
    return PyCryptoRSA.method("generate")
