from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class CryptoDSA(QueryType):
    fqns = ["cryptography.hazmat.primitives.asymmetric.dsa"]

class PyCryptoDSA(QueryType):
    fqns = ["Crypto.PublicKey.DSA", "Cryptodome.PublicKey.DSA"]


@python_rule(
    id="PYTHON-CRYPTO-SEC-024",
    name="DSA Key Generation (PyCryptodome, Audit)",
    severity="MEDIUM",
    category="cryptography",
    cwe="CWE-326",
    tags="python,pycryptodome,dsa,key-size,audit,cwe-326",
    message="DSA key generation detected. Ensure bits >= 2048.",
    owasp="A02:2021",
)
def detect_dsa_keygen_pycrypto():
    """Audit: detects DSA key generation in PyCryptodome."""
    return PyCryptoDSA.method("generate")
