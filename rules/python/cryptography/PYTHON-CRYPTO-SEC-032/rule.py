from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class PyCryptoAES(QueryType):
    fqns = ["Crypto.Cipher.AES", "Cryptodome.Cipher.AES"]


@python_rule(
    id="PYTHON-CRYPTO-SEC-032",
    name="Unauthenticated Cipher Mode (PyCryptodome)",
    severity="MEDIUM",
    category="cryptography",
    cwe="CWE-327",
    tags="python,pycryptodome,cipher-mode,unauthenticated,CWE-327",
    message="AES in non-GCM/EAX/SIV mode may lack authentication. Use MODE_GCM or MODE_EAX.",
    owasp="A02:2021",
)
def detect_aes_pycrypto_audit():
    """Audit: detects AES.new() usage in PyCryptodome for mode review."""
    return PyCryptoAES.method("new")
