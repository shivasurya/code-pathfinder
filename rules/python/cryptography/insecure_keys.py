"""
Python Insufficient Key Size Rules

PYTHON-CRYPTO-SEC-020: RSA Key Generation (cryptography) - audit
PYTHON-CRYPTO-SEC-021: DSA Key Generation (cryptography) - audit
PYTHON-CRYPTO-SEC-022: Weak EC Curve (cryptography) - audit
PYTHON-CRYPTO-SEC-023: RSA Key Generation (PyCryptodome) - audit
PYTHON-CRYPTO-SEC-024: DSA Key Generation (PyCryptodome) - audit

Security Impact: MEDIUM
CWE: CWE-326 (Inadequate Encryption Strength)
OWASP: A02:2021 - Cryptographic Failures

DESCRIPTION:
These rules audit asymmetric key generation operations across Python cryptographic libraries
(`cryptography` and `PyCryptodome`) to flag instances where key sizes may be insufficient.
They detect RSA, DSA, and Elliptic Curve key generation calls so developers can verify that
key sizes meet current security standards.

These are audit-level rules: they flag key generation for manual review rather than
definitively identifying a vulnerability, since the key size parameter requires value analysis
to determine if it is adequate.

Minimum recommended key sizes:
- **RSA**: 2048 bits minimum, 3072+ bits recommended (NIST SP 800-57)
- **DSA**: 2048 bits minimum (NIST SP 800-57)
- **Elliptic Curve**: 224 bits minimum; avoid SECP192R1 (NIST P-192)

SECURITY IMPLICATIONS:

**1. Brute-Force Key Recovery**:
Keys smaller than recommended sizes can be factored or solved with sufficient computational
resources. RSA-1024 is considered factorable by well-funded adversaries.

**2. Future-Proofing**:
Data encrypted today may be stored and decrypted in the future when computational power
increases. NIST recommends 3072-bit RSA for protection beyond 2030.

**3. Compliance Failures**:
Many compliance frameworks mandate minimum key sizes. Using undersized keys can result
in audit failures and regulatory penalties.

VULNERABLE EXAMPLE:
```python
from cryptography.hazmat.primitives.asymmetric import rsa, dsa, ec
from cryptography.hazmat.backends import default_backend

# SEC-020: RSA key generation (audit)
private_key = rsa.generate_private_key(
    public_exponent=65537,
    key_size=1024,
    backend=default_backend()
)

# SEC-021: DSA key generation (audit)
dsa_key = dsa.generate_private_key(
    key_size=1024,
    backend=default_backend()
)

# SEC-022: EC key generation (audit)
ec_key = ec.generate_private_key(
    ec.SECP192R1(),
    backend=default_backend()
)

# SEC-023: RSA in PyCryptodome (audit)
from Crypto.PublicKey import RSA
rsa_key = RSA.generate(1024)

# SEC-024: DSA in PyCryptodome (audit)
from Crypto.PublicKey import DSA
dsa_key_pc = DSA.generate(1024)
```

SECURE EXAMPLE:
```python
from cryptography.hazmat.primitives.asymmetric import rsa, dsa, ec
from cryptography.hazmat.backends import default_backend

# SECURE: RSA-3072 or higher
private_key = rsa.generate_private_key(
    public_exponent=65537,
    key_size=3072,
    backend=default_backend()
)

# SECURE: DSA-2048 or higher
private_key = dsa.generate_private_key(
    key_size=2048,
    backend=default_backend()
)

# SECURE: Strong elliptic curve (P-256 or higher)
private_key = ec.generate_private_key(
    ec.SECP256R1(),  # NIST P-256
    backend=default_backend()
)

# SECURE: PyCryptodome RSA-3072
from Crypto.PublicKey import RSA
key = RSA.generate(3072)
```

DETECTION AND PREVENTION:
```bash
# Audit key generation calls
pathfinder scan --project . --ruleset cpf/python/PYTHON-CRYPTO-SEC-020

# CI/CD integration
- name: Audit cryptographic key sizes
  run: pathfinder ci --project . --ruleset cpf/python/cryptography
```

**Code Review Checklist**:
- [ ] RSA key sizes are at least 2048 bits (3072+ preferred)
- [ ] DSA key sizes are at least 2048 bits
- [ ] Elliptic curve sizes are at least 224 bits (256+ preferred)
- [ ] No use of SECP192R1 (NIST P-192) curve
- [ ] Key sizes are appropriate for the data lifetime (longer-lived data needs larger keys)

COMPLIANCE:
- NIST SP 800-57: Minimum key size recommendations by algorithm type
- NIST SP 800-131A: RSA/DSA < 2048 bits disallowed after 2013
- PCI DSS: Strong cryptography with industry-accepted key lengths
- ANSSI (French): RSA >= 2048, EC >= 256 bits

REFERENCES:
- CWE-326: Inadequate Encryption Strength (https://cwe.mitre.org/data/definitions/326.html)
- NIST SP 800-57 Part 1 Rev 5: Key Management Recommendations
- NIST SP 800-131A Rev 2: Transitioning the Use of Cryptographic Algorithms
- OWASP Cryptographic Failures (https://owasp.org/Top10/A02_2021-Cryptographic_Failures/)
"""

from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets


class CryptoRSA(QueryType):
    fqns = ["cryptography.hazmat.primitives.asymmetric.rsa"]


class CryptoDSA(QueryType):
    fqns = ["cryptography.hazmat.primitives.asymmetric.dsa"]


class CryptoEC(QueryType):
    fqns = ["cryptography.hazmat.primitives.asymmetric.ec"]


class PyCryptoRSA(QueryType):
    fqns = ["Crypto.PublicKey.RSA", "Cryptodome.PublicKey.RSA"]


class PyCryptoDSA(QueryType):
    fqns = ["Crypto.PublicKey.DSA", "Cryptodome.PublicKey.DSA"]


@python_rule(
    id="PYTHON-CRYPTO-SEC-020",
    name="RSA Key Generation (Audit)",
    severity="MEDIUM",
    category="cryptography",
    cwe="CWE-326",
    tags="python,cryptography,rsa,key-size,audit,cwe-326",
    message="RSA key generation detected. Ensure key_size >= 2048 bits.",
    owasp="A02:2021",
)
def detect_rsa_keygen_crypto():
    """Audit: detects RSA key generation in cryptography lib."""
    return CryptoRSA.method("generate_private_key")


@python_rule(
    id="PYTHON-CRYPTO-SEC-021",
    name="DSA Key Generation (Audit)",
    severity="MEDIUM",
    category="cryptography",
    cwe="CWE-326",
    tags="python,cryptography,dsa,key-size,audit,cwe-326",
    message="DSA key generation detected. Ensure key_size >= 2048 bits.",
    owasp="A02:2021",
)
def detect_dsa_keygen_crypto():
    """Audit: detects DSA key generation in cryptography lib."""
    return CryptoDSA.method("generate_private_key")


@python_rule(
    id="PYTHON-CRYPTO-SEC-022",
    name="EC Key Generation (Audit)",
    severity="LOW",
    category="cryptography",
    cwe="CWE-326",
    tags="python,cryptography,ec,elliptic-curve,key-size,audit,cwe-326",
    message="EC key generation detected. Ensure curve >= 224 bits (avoid SECP192R1).",
    owasp="A02:2021",
)
def detect_ec_keygen_crypto():
    """Audit: detects EC key generation in cryptography lib."""
    return CryptoEC.method("generate_private_key")


@python_rule(
    id="PYTHON-CRYPTO-SEC-023",
    name="RSA Key Generation (PyCryptodome, Audit)",
    severity="MEDIUM",
    category="cryptography",
    cwe="CWE-326",
    tags="python,pycryptodome,rsa,key-size,audit,cwe-326",
    message="RSA key generation detected. Ensure bits >= 3072.",
    owasp="A02:2021",
)
def detect_rsa_keygen_pycrypto():
    """Audit: detects RSA key generation in PyCryptodome."""
    return PyCryptoRSA.method("generate")


@python_rule(
    id="PYTHON-CRYPTO-SEC-024",
    name="DSA Key Generation (PyCryptodome, Audit)",
    severity="MEDIUM",
    category="cryptography",
    cwe="CWE-326",
    tags="python,pycryptodome,dsa,key-size,audit,cwe-326",
    message="DSA key generation detected. Ensure bits >= 2048.",
    owasp="A02:2021",
)
def detect_dsa_keygen_pycrypto():
    """Audit: detects DSA key generation in PyCryptodome."""
    return PyCryptoDSA.method("generate")
