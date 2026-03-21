from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class PyCryptoHashSHA(QueryType):
    fqns = ["Crypto.Hash.SHA", "Cryptodome.Hash.SHA",
            "Crypto.Hash.SHA1", "Cryptodome.Hash.SHA1"]


@python_rule(
    id="PYTHON-CRYPTO-SEC-015",
    name="Insecure SHA1 Hash (PyCryptodome)",
    severity="MEDIUM",
    category="cryptography",
    cwe="CWE-327",
    tags="python,pycryptodome,sha1,weak-hash,cwe-327",
    message="SHA-1 is deprecated for security use. Use SHA-256 or SHA-3 instead.",
    owasp="A02:2021",
)
def detect_sha1_hash_pycrypto():
    """Detects SHA1 in PyCryptodome."""
    return PyCryptoHashSHA.method("new")
