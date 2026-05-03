from codepathfinder.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class CryptoCipherAlgorithms(QueryType):
    fqns = ["cryptography.hazmat.primitives.ciphers.algorithms"]


@python_rule(
    id="PYTHON-CRYPTO-SEC-002",
    name="Insecure Blowfish Cipher",
    severity="HIGH",
    category="cryptography",
    cwe="CWE-327",
    tags="python,cryptography,blowfish,weak-cipher,CWE-327",
    message="Blowfish has a 64-bit block size vulnerable to birthday attacks. Use AES instead.",
    owasp="A02:2021",
)
def detect_blowfish_cipher():
    """Detects Blowfish cipher usage in cryptography lib."""
    return CryptoCipherAlgorithms.method("Blowfish")
