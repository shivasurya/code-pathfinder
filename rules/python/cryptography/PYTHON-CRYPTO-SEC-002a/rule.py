from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class PyCryptoCipherBlowfish(QueryType):
    fqns = ["Crypto.Cipher.Blowfish", "Cryptodome.Cipher.Blowfish"]


@python_rule(
    id="PYTHON-CRYPTO-SEC-002a",
    name="Insecure Blowfish Cipher (PyCryptodome)",
    severity="HIGH",
    category="cryptography",
    cwe="CWE-327",
    tags="python,pycryptodome,blowfish,weak-cipher,cwe-327",
    message="Blowfish has a 64-bit block size vulnerable to birthday attacks. Use AES instead.",
    owasp="A02:2021",
)
def detect_blowfish_cipher_pycrypto():
    """Detects Blowfish in PyCryptodome."""
    return PyCryptoCipherBlowfish.method("new")
