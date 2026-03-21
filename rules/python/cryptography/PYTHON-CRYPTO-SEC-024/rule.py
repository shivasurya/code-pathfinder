from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.qualifiers import lt


class PyCryptoDSA(QueryType):
    fqns = ["Crypto.PublicKey.DSA", "Cryptodome.PublicKey.DSA"]


@python_rule(
    id="PYTHON-CRYPTO-SEC-024",
    name="Insufficient DSA Key Size (PyCryptodome)",
    severity="HIGH",
    category="cryptography",
    cwe="CWE-326",
    tags="python,pycryptodome,dsa,key-size,CWE-326,OWASP-A02",
    message="DSA key size is less than 2048 bits.",
    owasp="A02:2021",
)
def detect_dsa_keygen_pycrypto():
    """Detects DSA key generation with insufficient key size in PyCryptodome."""
    return PyCryptoDSA.method("generate").where(0, lt(2048))
