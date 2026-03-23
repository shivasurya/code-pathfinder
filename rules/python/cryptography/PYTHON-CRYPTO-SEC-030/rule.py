from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class CryptoModes(QueryType):
    fqns = ["cryptography.hazmat.primitives.ciphers.modes"]


@python_rule(
    id="PYTHON-CRYPTO-SEC-030",
    name="ECB Mode Usage",
    severity="HIGH",
    category="cryptography",
    cwe="CWE-327",
    tags="python,cryptography,ecb,cipher-mode,weak-mode,CWE-327",
    message="ECB mode does not provide semantic security. Use CBC, CTR, or GCM instead.",
    owasp="A02:2021",
)
def detect_ecb_mode():
    """Detects ECB cipher mode in cryptography lib."""
    return CryptoModes.method("ECB")
