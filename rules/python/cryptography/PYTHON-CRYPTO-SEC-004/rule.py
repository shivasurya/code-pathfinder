from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class PyCryptoCipherARC2(QueryType):
    fqns = ["Crypto.Cipher.ARC2", "Cryptodome.Cipher.ARC2"]


@python_rule(
    id="PYTHON-CRYPTO-SEC-004",
    name="Insecure RC2 Cipher (PyCryptodome)",
    severity="HIGH",
    category="cryptography",
    cwe="CWE-327",
    tags="python,pycryptodome,rc2,weak-cipher,cwe-327",
    message="RC2 cipher is weak. Use AES instead.",
    owasp="A02:2021",
)
def detect_rc2_cipher():
    """Detects RC2/ARC2 cipher in PyCryptodome."""
    return PyCryptoCipherARC2.method("new")
