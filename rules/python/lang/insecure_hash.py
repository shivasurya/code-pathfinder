"""
Insecure Cryptographic Hash Algorithm Security Rules for Python

Rules in this file:
- PYTHON-LANG-SEC-030: Insecure MD5 Hash Usage (CWE-327)
- PYTHON-LANG-SEC-031: Insecure SHA1 Hash Usage (CWE-327)
- PYTHON-LANG-SEC-032: Insecure Hash via hashlib.new() (CWE-327)
- PYTHON-LANG-SEC-033: SHA-224 Weak Hash (CWE-327)
- PYTHON-LANG-SEC-034: MD5 Used for Password Hashing (CWE-327)

Security Impact: MEDIUM
CWE: CWE-327 (Use of a Broken or Risky Cryptographic Algorithm)
OWASP: A02:2021 - Cryptographic Failures

DESCRIPTION:
MD5 and SHA-1 are cryptographically broken hash algorithms with known collision
attacks. MD5 collisions can be generated in seconds on modern hardware, and
SHA-1 collisions have been demonstrated practically (SHAttered attack, 2017).
Using these algorithms for integrity verification, digital signatures, certificate
validation, or password hashing provides a false sense of security and exposes
applications to forgery and pre-image attacks.

SECURITY IMPLICATIONS:
An attacker can forge documents, binaries, or certificates that produce identical
MD5 or SHA-1 hashes as legitimate files (collision attack). When MD5 is used for
password hashing, rainbow tables and GPU-accelerated brute-force attacks can
recover passwords in minutes. SHA-224 provides only 112-bit collision resistance,
which falls below the 128-bit minimum recommended by NIST for new applications.
The hashlib.new() function may be called with weak algorithm names passed as
runtime strings, making static detection harder.

    # Attack scenario: forged file with matching MD5
    # attacker generates collision: md5(malicious.pdf) == md5(legitimate.pdf)
    checksum = hashlib.md5(uploaded_file).hexdigest()
    if checksum == expected_checksum:  # Passes verification with forged file
        process_file(uploaded_file)

VULNERABLE EXAMPLE:
```python
import hashlib

# SEC-030: MD5
md5_hash = hashlib.md5(b"data")

# SEC-031: SHA1
sha1_hash = hashlib.sha1(b"data")

# SEC-032: hashlib.new with insecure algo
h = hashlib.new("md5", b"data")

# SEC-033: SHA224
sha224_hash = hashlib.sha224(b"data")
sha3_224_hash = hashlib.sha3_224(b"data")

# SEC-034: MD5 for password
def hash_password(password):
    return hashlib.md5(password.encode()).hexdigest()
```

SECURE EXAMPLE:
```python
import hashlib
# Use SHA-256 or SHA-3 for integrity verification
file_hash = hashlib.sha256(data).hexdigest()
# Use dedicated password hashing (bcrypt, scrypt, argon2)
import bcrypt
pwd_hash = bcrypt.hashpw(password.encode(), bcrypt.gensalt())
# SHA-3 for digital signatures
sig_hash = hashlib.sha3_256(document).hexdigest()
```

DETECTION AND PREVENTION:
- Replace MD5 and SHA-1 with SHA-256, SHA-384, SHA-512, or SHA-3 family
- Use dedicated password hashing functions: bcrypt, scrypt, argon2id, or PBKDF2
- Audit hashlib.new() calls to ensure algorithm names resolve to strong algorithms
- For non-security checksums (cache keys, deduplication), document the rationale
- Enforce minimum 128-bit security level for all cryptographic hash operations

COMPLIANCE:
- CWE-327: Use of a Broken or Risky Cryptographic Algorithm
- CWE-328: Use of Weak Hash
- OWASP A02:2021 - Cryptographic Failures
- NIST SP 800-131A: Transitioning the Use of Cryptographic Algorithms
- PCI DSS v4.0 Requirement 4.2: Strong cryptography for transmission
- FIPS 180-4: Secure Hash Standard (SHA-256, SHA-384, SHA-512)

REFERENCES:
- https://cwe.mitre.org/data/definitions/327.html
- https://owasp.org/Top10/A02_2021-Cryptographic_Failures/
- https://docs.python.org/3/library/hashlib.html
- https://shattered.io/ (SHA-1 collision demonstration)
- https://csrc.nist.gov/publications/detail/sp/800-131a/rev-2/final
"""

from rules.python_decorators import python_rule
from codepathfinder import calls, QueryType


class HashlibModule(QueryType):
    fqns = ["hashlib"]


@python_rule(
    id="PYTHON-LANG-SEC-030",
    name="Insecure MD5 Hash Usage",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-327",
    tags="python,md5,weak-hash,cryptography,owasp-a02,cwe-327",
    message="MD5 is cryptographically broken. Use SHA-256 or SHA-3 for security-sensitive hashing.",
    owasp="A02:2021",
)
def detect_md5():
    """Detects hashlib.md5() usage."""
    return HashlibModule.method("md5")


@python_rule(
    id="PYTHON-LANG-SEC-031",
    name="Insecure SHA1 Hash Usage",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-327",
    tags="python,sha1,weak-hash,cryptography,owasp-a02,cwe-327",
    message="SHA-1 is cryptographically weak. Use SHA-256 or SHA-3 instead.",
    owasp="A02:2021",
)
def detect_sha1():
    """Detects hashlib.sha1() usage."""
    return HashlibModule.method("sha1")


@python_rule(
    id="PYTHON-LANG-SEC-032",
    name="Insecure Hash via hashlib.new()",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-327",
    tags="python,weak-hash,hashlib-new,cwe-327",
    message="hashlib.new() with insecure algorithm. Use SHA-256 or SHA-3.",
    owasp="A02:2021",
)
def detect_hashlib_new_insecure():
    """Detects hashlib.new() which may use insecure algorithms."""
    return HashlibModule.method("new")


@python_rule(
    id="PYTHON-LANG-SEC-033",
    name="SHA-224 Weak Hash",
    severity="LOW",
    category="lang",
    cwe="CWE-327",
    tags="python,sha224,weak-hash,cwe-327",
    message="SHA-224 provides only 112-bit security. Consider SHA-256 or SHA-3.",
    owasp="A02:2021",
)
def detect_sha224():
    """Detects hashlib.sha224() and sha3_224() usage."""
    return HashlibModule.method("sha224", "sha3_224")


@python_rule(
    id="PYTHON-LANG-SEC-034",
    name="MD5 Used for Password Hashing",
    severity="HIGH",
    category="lang",
    cwe="CWE-327",
    tags="python,md5,password,weak-hash,cwe-327",
    message="MD5 used for password hashing. Use bcrypt, scrypt, or argon2 instead.",
    owasp="A02:2021",
)
def detect_md5_password():
    """Detects MD5 used in password context -- audit-level detection."""
    return HashlibModule.method("md5")
