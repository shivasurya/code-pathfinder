from codepathfinder.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class CryptoHashes(QueryType):
    fqns = ["cryptography.hazmat.primitives.hashes"]


@python_rule(
    id="PYTHON-CRYPTO-SEC-010",
    name="Insecure MD5 Hash (cryptography)",
    severity="MEDIUM",
    category="cryptography",
    cwe="CWE-327",
    tags="python,cryptography,md5,weak-hash,CWE-327",
    message="MD5 is cryptographically broken. Use SHA-256 or SHA-3 instead.",
    owasp="A02:2021",
)
def detect_md5_hash_crypto():
    """Detects MD5 usage in cryptography library."""
    return CryptoHashes.method("MD5")
