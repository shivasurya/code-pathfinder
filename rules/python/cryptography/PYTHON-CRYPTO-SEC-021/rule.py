from codepathfinder.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.qualifiers import lt


class CryptoDSA(QueryType):
    fqns = ["cryptography.hazmat.primitives.asymmetric.dsa"]


@python_rule(
    id="PYTHON-CRYPTO-SEC-021",
    name="Insufficient DSA Key Size",
    severity="HIGH",
    category="cryptography",
    cwe="CWE-326",
    tags="python,cryptography,dsa,key-size,CWE-326,OWASP-A02",
    message="DSA key size is less than 2048 bits. Use at least 2048-bit keys.",
    owasp="A02:2021",
)
def detect_dsa_keygen_crypto():
    """Detects DSA key generation with insufficient key size in cryptography lib."""
    return CryptoDSA.method("generate_private_key").where("key_size", lt(2048))
