from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets

class CryptoHashes(QueryType):
    fqns = ["cryptography.hazmat.primitives.hashes"]


@python_rule(
    id="PYTHON-CRYPTO-SEC-011",
    name="Insecure SHA1 Hash (cryptography)",
    severity="MEDIUM",
    category="cryptography",
    cwe="CWE-327",
    tags="python,cryptography,sha1,weak-hash,CWE-327",
    message="SHA-1 is deprecated for security use. Use SHA-256 or SHA-3 instead.",
    owasp="A02:2021",
)
def detect_sha1_hash_crypto():
    """Detects SHA1 usage in cryptography library."""
    return CryptoHashes.method("SHA1")
