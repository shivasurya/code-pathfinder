from codepathfinder.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.qualifiers import lt


class PyCryptoRSA(QueryType):
    fqns = ["Crypto.PublicKey.RSA", "Cryptodome.PublicKey.RSA"]


@python_rule(
    id="PYTHON-CRYPTO-SEC-023",
    name="Insufficient RSA Key Size (PyCryptodome)",
    severity="HIGH",
    category="cryptography",
    cwe="CWE-326",
    tags="python,pycryptodome,rsa,key-size,CWE-326,OWASP-A02",
    message="RSA key size is less than 3072 bits. PyCryptodome recommends 3072+ bit keys.",
    owasp="A02:2021",
)
def detect_rsa_keygen_pycrypto():
    """Detects RSA key generation with insufficient key size in PyCryptodome."""
    return PyCryptoRSA.method("generate").where(0, lt(3072))
