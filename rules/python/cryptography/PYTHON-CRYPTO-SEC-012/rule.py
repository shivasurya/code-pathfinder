from codepathfinder.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class PyCryptoHashMD5(QueryType):
    fqns = ["Crypto.Hash.MD5", "Cryptodome.Hash.MD5"]


@python_rule(
    id="PYTHON-CRYPTO-SEC-012",
    name="Insecure MD5 Hash (PyCryptodome)",
    severity="MEDIUM",
    category="cryptography",
    cwe="CWE-327",
    tags="python,pycryptodome,md5,weak-hash,CWE-327",
    message="MD5 is cryptographically broken. Use SHA-256 or SHA-3 instead.",
    owasp="A02:2021",
)
def detect_md5_hash_pycrypto():
    """Detects MD5 in PyCryptodome."""
    return PyCryptoHashMD5.method("new")
