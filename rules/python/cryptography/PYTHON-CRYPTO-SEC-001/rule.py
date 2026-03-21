from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class CryptoCipherAlgorithms(QueryType):
    fqns = ["cryptography.hazmat.primitives.ciphers.algorithms"]


@python_rule(
    id="PYTHON-CRYPTO-SEC-001",
    name="Insecure ARC4 (RC4) Cipher",
    severity="HIGH",
    category="cryptography",
    cwe="CWE-327",
    tags="python,cryptography,arc4,rc4,weak-cipher,cwe-327",
    message="ARC4/RC4 is a broken stream cipher. Use AES-GCM or ChaCha20Poly1305 instead.",
    owasp="A02:2021",
)
def detect_arc4_cipher():
    """Detects ARC4 cipher usage in cryptography and pycryptodome."""
    return CryptoCipherAlgorithms.method("ARC4")
