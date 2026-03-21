from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class PyCryptoCipherARC4(QueryType):
    fqns = ["Crypto.Cipher.ARC4", "Cryptodome.Cipher.ARC4"]


@python_rule(
    id="PYTHON-CRYPTO-SEC-001a",
    name="Insecure ARC4 (RC4) Cipher (PyCryptodome)",
    severity="HIGH",
    category="cryptography",
    cwe="CWE-327",
    tags="python,pycryptodome,arc4,rc4,weak-cipher,cwe-327",
    message="ARC4/RC4 is a broken stream cipher. Use AES-GCM or ChaCha20Poly1305 instead.",
    owasp="A02:2021",
)
def detect_arc4_cipher_pycrypto():
    """Detects ARC4 in PyCryptodome."""
    return PyCryptoCipherARC4.method("new")
