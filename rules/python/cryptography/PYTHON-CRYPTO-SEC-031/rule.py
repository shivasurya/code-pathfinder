from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class CryptoModes(QueryType):
    fqns = ["cryptography.hazmat.primitives.ciphers.modes"]


@python_rule(
    id="PYTHON-CRYPTO-SEC-031",
    name="Unauthenticated Cipher Mode (cryptography)",
    severity="MEDIUM",
    category="cryptography",
    cwe="CWE-327",
    tags="python,cryptography,cipher-mode,unauthenticated,cwe-327",
    message="CBC/CTR/CFB/OFB without authentication (HMAC). Use GCM or add HMAC.",
    owasp="A02:2021",
)
def detect_unauthenticated_mode_crypto():
    """Audit: detects CBC/CTR/CFB/OFB mode usage that may lack authentication."""
    return CryptoModes.method("CBC", "CTR", "CFB", "OFB")
