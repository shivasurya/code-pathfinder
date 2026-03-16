from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class PyCryptoCipherXOR(QueryType):
    fqns = ["Crypto.Cipher.XOR", "Cryptodome.Cipher.XOR"]


@python_rule(
    id="PYTHON-CRYPTO-SEC-006",
    name="Insecure XOR Cipher (PyCryptodome)",
    severity="HIGH",
    category="cryptography",
    cwe="CWE-327",
    tags="python,pycryptodome,xor,weak-cipher,cwe-327",
    message="XOR cipher provides no real security. Use AES instead.",
    owasp="A02:2021",
)
def detect_xor_cipher():
    """Detects XOR cipher in PyCryptodome."""
    return PyCryptoCipherXOR.method("new")
