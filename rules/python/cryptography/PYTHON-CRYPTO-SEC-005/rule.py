from codepathfinder.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class PyCryptoCipherDES(QueryType):
    fqns = ["Crypto.Cipher.DES", "Cryptodome.Cipher.DES"]


@python_rule(
    id="PYTHON-CRYPTO-SEC-005",
    name="Insecure DES Cipher",
    severity="HIGH",
    category="cryptography",
    cwe="CWE-327",
    tags="python,pycryptodome,des,weak-cipher,CWE-327",
    message="DES has a 56-bit key, easily brute-forced. Use AES instead.",
    owasp="A02:2021",
)
def detect_des_cipher():
    """Detects DES cipher in PyCryptodome."""
    return PyCryptoCipherDES.method("new")
