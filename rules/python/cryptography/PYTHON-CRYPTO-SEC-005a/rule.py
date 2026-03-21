from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class PyCryptoCipherDES(QueryType):
    fqns = ["Crypto.Cipher.DES", "Cryptodome.Cipher.DES"]

class PyCryptoCipherDES3(QueryType):
    fqns = ["Crypto.Cipher.DES3", "Cryptodome.Cipher.DES3"]


@python_rule(
    id="PYTHON-CRYPTO-SEC-005a",
    name="Insecure Triple DES Cipher",
    severity="MEDIUM",
    category="cryptography",
    cwe="CWE-327",
    tags="python,pycryptodome,3des,triple-des,weak-cipher,cwe-327",
    message="Triple DES (3DES) is deprecated. Use AES instead.",
    owasp="A02:2021",
)
def detect_des3_cipher():
    """Detects Triple DES cipher in PyCryptodome."""
    return PyCryptoCipherDES3.method("new")
