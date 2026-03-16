from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class PyCryptoHashMD4(QueryType):
    fqns = ["Crypto.Hash.MD4", "Cryptodome.Hash.MD4"]


@python_rule(
    id="PYTHON-CRYPTO-SEC-013",
    name="Insecure MD4 Hash (PyCryptodome)",
    severity="HIGH",
    category="cryptography",
    cwe="CWE-327",
    tags="python,pycryptodome,md4,weak-hash,cwe-327",
    message="MD4 is severely broken. Use SHA-256 or SHA-3 instead.",
    owasp="A02:2021",
)
def detect_md4_hash_pycrypto():
    """Detects MD4 in PyCryptodome."""
    return PyCryptoHashMD4.method("new")
