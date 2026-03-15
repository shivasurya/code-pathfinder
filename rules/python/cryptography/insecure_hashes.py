"""
Python Insecure Hash Algorithm Rules

PYTHON-CRYPTO-SEC-010: Insecure MD5 Hash (cryptography lib)
PYTHON-CRYPTO-SEC-011: Insecure SHA1 Hash (cryptography lib)
PYTHON-CRYPTO-SEC-012: Insecure MD5 Hash (PyCryptodome)
PYTHON-CRYPTO-SEC-013: Insecure MD4 Hash (PyCryptodome)
PYTHON-CRYPTO-SEC-014: Insecure MD2 Hash (PyCryptodome)
PYTHON-CRYPTO-SEC-015: Insecure SHA1 Hash (PyCryptodome)

Security Impact: MEDIUM to HIGH
CWE: CWE-327 (Use of a Broken or Risky Cryptographic Algorithm)
OWASP: A02:2021 - Cryptographic Failures

DESCRIPTION:
These rules detect usage of cryptographically broken or weak hash algorithms across
Python libraries including `cryptography` (hazmat primitives) and `PyCryptodome`/`PyCrypto`.
Weak hash functions are vulnerable to collision attacks and should not be used for security
purposes such as digital signatures, certificate validation, password hashing, or data
integrity verification.

Detected insecure hash algorithms:
- **MD2**: Severely broken; collision attacks demonstrated with practical complexity
- **MD4**: Severely broken; collisions can be found in seconds on consumer hardware
- **MD5**: Collision attacks fully practical (CVE-2004-2761); used in Flame malware to forge certificates
- **SHA-1**: Collision demonstrated by SHAttered attack (2017); NIST deprecated for digital signatures

SECURITY IMPLICATIONS:

**1. Collision Attacks**:
An attacker can create two different inputs that produce the same hash, allowing them to
substitute malicious content for legitimate content without detection.

**2. Certificate Forgery**:
MD5 collisions were used in real-world attacks to forge CA certificates (2008 Rogue CA attack).
SHA-1 collisions make certificate forgery feasible for well-funded attackers.

**3. Integrity Bypass**:
If file integrity or message authentication relies on MD5/SHA-1, an attacker can modify
data while preserving the hash value.

**4. Password Cracking**:
MD5 and SHA-1 are extremely fast, making brute-force and rainbow table attacks efficient.
Use bcrypt, scrypt, or Argon2 for password hashing instead.

VULNERABLE EXAMPLE:
```python
from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.backends import default_backend

# SEC-010: MD5 in cryptography lib
digest = hashes.Hash(hashes.MD5(), backend=default_backend())

# SEC-011: SHA1 in cryptography lib
digest_sha1 = hashes.Hash(hashes.SHA1(), backend=default_backend())

# SEC-012: MD5 in PyCryptodome
from Crypto.Hash import MD5
h_md5 = MD5.new(b"data")

# SEC-013: MD4 in PyCryptodome
from Crypto.Hash import MD4
h_md4 = MD4.new(b"data")

# SEC-014: MD2 in PyCryptodome
from Crypto.Hash import MD2
h_md2 = MD2.new(b"data")

# SEC-015: SHA1 in PyCryptodome
from Crypto.Hash import SHA
h_sha = SHA.new(b"data")
```

SECURE EXAMPLE:
```python
from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.backends import default_backend

# SECURE: SHA-256 for integrity checking
digest = hashes.Hash(hashes.SHA256(), backend=default_backend())
digest.update(document_bytes)
checksum = digest.finalize()

# SECURE: SHA-3 for digital signatures
from Crypto.Hash import SHA3_256
h = SHA3_256.new()
h.update(b"data to sign")
hash_value = h.hexdigest()

# SECURE: Argon2 for password hashing
from argon2 import PasswordHasher
ph = PasswordHasher()
password_hash = ph.hash(password)
```

DETECTION AND PREVENTION:
```bash
# Scan for insecure hash usage
pathfinder scan --project . --ruleset cpf/python/PYTHON-CRYPTO-SEC-010

# CI/CD integration
- name: Check for insecure hashes
  run: pathfinder ci --project . --ruleset cpf/python/cryptography
```

**Code Review Checklist**:
- [ ] No MD2, MD4, MD5, or SHA-1 used for security purposes
- [ ] All hashing uses SHA-256, SHA-384, SHA-512, or SHA-3
- [ ] Password hashing uses bcrypt, scrypt, or Argon2 (not raw hash functions)
- [ ] HMAC uses SHA-256 or stronger as the underlying hash

**Note**: MD5/SHA-1 may be acceptable for non-security uses such as checksums for data
deduplication or cache keys where collision resistance is not required.

COMPLIANCE:
- NIST SP 800-131A: MD5 and SHA-1 disallowed for digital signatures
- PCI DSS: Strong cryptography required for cardholder data protection
- FIPS 180-4: SHA-1 deprecated; SHA-2 and SHA-3 families approved
- CA/Browser Forum: SHA-1 certificates prohibited since January 2017

REFERENCES:
- CWE-327: Use of a Broken or Risky Cryptographic Algorithm (https://cwe.mitre.org/data/definitions/327.html)
- CVE-2004-2761: MD5 collision vulnerability in certificate signing
- SHAttered: SHA-1 collision attack (https://shattered.io/)
- NIST SP 800-131A Rev 2: Transitioning the Use of Cryptographic Algorithms
- OWASP Cryptographic Failures (https://owasp.org/Top10/A02_2021-Cryptographic_Failures/)
"""

from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets


class CryptoHashes(QueryType):
    fqns = ["cryptography.hazmat.primitives.hashes"]


class PyCryptoHashMD5(QueryType):
    fqns = ["Crypto.Hash.MD5", "Cryptodome.Hash.MD5"]


class PyCryptoHashMD4(QueryType):
    fqns = ["Crypto.Hash.MD4", "Cryptodome.Hash.MD4"]


class PyCryptoHashMD2(QueryType):
    fqns = ["Crypto.Hash.MD2", "Cryptodome.Hash.MD2"]


class PyCryptoHashSHA(QueryType):
    fqns = ["Crypto.Hash.SHA", "Cryptodome.Hash.SHA",
            "Crypto.Hash.SHA1", "Cryptodome.Hash.SHA1"]


@python_rule(
    id="PYTHON-CRYPTO-SEC-010",
    name="Insecure MD5 Hash (cryptography)",
    severity="MEDIUM",
    category="cryptography",
    cwe="CWE-327",
    tags="python,cryptography,md5,weak-hash,cwe-327",
    message="MD5 is cryptographically broken. Use SHA-256 or SHA-3 instead.",
    owasp="A02:2021",
)
def detect_md5_hash_crypto():
    """Detects MD5 usage in cryptography library."""
    return CryptoHashes.method("MD5")


@python_rule(
    id="PYTHON-CRYPTO-SEC-011",
    name="Insecure SHA1 Hash (cryptography)",
    severity="MEDIUM",
    category="cryptography",
    cwe="CWE-327",
    tags="python,cryptography,sha1,weak-hash,cwe-327",
    message="SHA-1 is deprecated for security use. Use SHA-256 or SHA-3 instead.",
    owasp="A02:2021",
)
def detect_sha1_hash_crypto():
    """Detects SHA1 usage in cryptography library."""
    return CryptoHashes.method("SHA1")


@python_rule(
    id="PYTHON-CRYPTO-SEC-012",
    name="Insecure MD5 Hash (PyCryptodome)",
    severity="MEDIUM",
    category="cryptography",
    cwe="CWE-327",
    tags="python,pycryptodome,md5,weak-hash,cwe-327",
    message="MD5 is cryptographically broken. Use SHA-256 or SHA-3 instead.",
    owasp="A02:2021",
)
def detect_md5_hash_pycrypto():
    """Detects MD5 in PyCryptodome."""
    return PyCryptoHashMD5.method("new")


@python_rule(
    id="PYTHON-CRYPTO-SEC-013",
    name="Insecure MD4 Hash (PyCryptodome)",
    severity="HIGH",
    category="cryptography",
    cwe="CWE-327",
    tags="python,pycryptodome,md4,weak-hash,cwe-327",
    message="MD4 is severely broken. Use SHA-256 or SHA-3 instead.",
    owasp="A02:2021",
)
def detect_md4_hash_pycrypto():
    """Detects MD4 in PyCryptodome."""
    return PyCryptoHashMD4.method("new")


@python_rule(
    id="PYTHON-CRYPTO-SEC-014",
    name="Insecure MD2 Hash (PyCryptodome)",
    severity="HIGH",
    category="cryptography",
    cwe="CWE-327",
    tags="python,pycryptodome,md2,weak-hash,cwe-327",
    message="MD2 is severely broken. Use SHA-256 or SHA-3 instead.",
    owasp="A02:2021",
)
def detect_md2_hash_pycrypto():
    """Detects MD2 in PyCryptodome."""
    return PyCryptoHashMD2.method("new")


@python_rule(
    id="PYTHON-CRYPTO-SEC-015",
    name="Insecure SHA1 Hash (PyCryptodome)",
    severity="MEDIUM",
    category="cryptography",
    cwe="CWE-327",
    tags="python,pycryptodome,sha1,weak-hash,cwe-327",
    message="SHA-1 is deprecated for security use. Use SHA-256 or SHA-3 instead.",
    owasp="A02:2021",
)
def detect_sha1_hash_pycrypto():
    """Detects SHA1 in PyCryptodome."""
    return PyCryptoHashSHA.method("new")
