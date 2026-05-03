from codepathfinder.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class PyCryptoHashMD2(QueryType):
    fqns = ["Crypto.Hash.MD2", "Cryptodome.Hash.MD2"]


@python_rule(
    id="PYTHON-CRYPTO-SEC-014",
    name="Insecure MD2 Hash (PyCryptodome)",
    severity="HIGH",
    category="cryptography",
    cwe="CWE-327",
    tags="python,pycryptodome,md2,weak-hash,CWE-327",
    message="MD2 is severely broken. Use SHA-256 or SHA-3 instead.",
    owasp="A02:2021",
)
def detect_md2_hash_pycrypto():
    """Detects MD2 in PyCryptodome."""
    return PyCryptoHashMD2.method("new")
