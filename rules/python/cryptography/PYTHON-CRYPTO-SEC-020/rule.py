from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.qualifiers import lt


class CryptoRSA(QueryType):
    fqns = ["cryptography.hazmat.primitives.asymmetric.rsa"]


@python_rule(
    id="PYTHON-CRYPTO-SEC-020",
    name="Insufficient RSA Key Size",
    severity="HIGH",
    category="cryptography",
    cwe="CWE-326",
    tags="python,cryptography,rsa,key-size,CWE-326,OWASP-A02",
    message="RSA key size is less than 2048 bits. Use at least 2048-bit keys (3072+ recommended).",
    owasp="A02:2021",
)
def detect_rsa_keygen_crypto():
    """Detects RSA key generation with insufficient key size in cryptography lib."""
    return CryptoRSA.method("generate_private_key").where("key_size", lt(2048))
