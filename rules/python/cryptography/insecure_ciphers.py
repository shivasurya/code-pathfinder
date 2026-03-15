"""
Python Insecure Cipher Algorithm Rules

PYTHON-CRYPTO-SEC-001: Insecure ARC4 (RC4) Cipher
PYTHON-CRYPTO-SEC-002: Insecure Blowfish Cipher
PYTHON-CRYPTO-SEC-003: Insecure IDEA Cipher
PYTHON-CRYPTO-SEC-004: Insecure RC2 Cipher
PYTHON-CRYPTO-SEC-005: Insecure DES Cipher
PYTHON-CRYPTO-SEC-006: Insecure XOR Cipher

Security Impact: HIGH
CWE: CWE-327 (Use of a Broken or Risky Cryptographic Algorithm)
OWASP: A02:2021 - Cryptographic Failures

DESCRIPTION:
These rules detect usage of insecure or deprecated symmetric cipher algorithms across
Python cryptographic libraries including `cryptography` (hazmat primitives) and
`PyCryptodome`/`PyCrypto`. The flagged ciphers have known cryptographic weaknesses that
make them unsuitable for protecting sensitive data.

Detected insecure ciphers:
- **ARC4/RC4**: Broken stream cipher with biases in keystream output (RFC 7465 prohibits use in TLS)
- **Blowfish**: 64-bit block size vulnerable to birthday attacks (Sweet32, CVE-2016-2183)
- **IDEA**: Deprecated cipher removed from modern standards
- **RC2**: Weak key schedule and small effective key size
- **DES**: 56-bit key trivially brute-forced (broken since 1999)
- **Triple DES (3DES)**: Deprecated due to Sweet32 attacks; limited to 112-bit effective security
- **XOR**: Not a real cipher; provides zero cryptographic security

SECURITY IMPLICATIONS:

**1. Data Exposure**:
Encrypted data protected by weak ciphers can be decrypted by attackers through known
cryptanalytic attacks, brute force, or statistical analysis of ciphertext.

**2. Compliance Violations**:
NIST, PCI DSS, HIPAA, and other standards explicitly prohibit or deprecate these algorithms
for protecting sensitive data.

**3. Birthday Attacks (64-bit block ciphers)**:
Blowfish, DES, and 3DES use 64-bit blocks. After approximately 2^32 blocks (~32 GB) of data
encrypted under the same key, collisions allow plaintext recovery (Sweet32 attack).

**4. Key Recovery**:
DES and RC2 have small key spaces that are trivially searchable with modern hardware.
ARC4 has biases in its keystream that leak information about the key.

VULNERABLE EXAMPLE:
```python
from cryptography.hazmat.primitives.ciphers import Cipher, algorithms, modes
from cryptography.hazmat.backends import default_backend

# VULNERABLE: Using ARC4 stream cipher
key = os.urandom(16)
cipher = Cipher(algorithms.ARC4(key), mode=None, backend=default_backend())
encryptor = cipher.encryptor()
ct = encryptor.update(b"sensitive data")

# VULNERABLE: Using DES with PyCryptodome
from Crypto.Cipher import DES
key = b'8bytekey'
cipher = DES.new(key, DES.MODE_ECB)
ct = cipher.encrypt(b"secret!!")

# VULNERABLE: Using Blowfish
from Crypto.Cipher import Blowfish
cipher = Blowfish.new(key, Blowfish.MODE_CBC)
```

SECURE EXAMPLE:
```python
from cryptography.hazmat.primitives.ciphers.aead import AESGCM
import os

# SECURE: AES-256-GCM provides both confidentiality and integrity
key = AESGCM.generate_key(bit_length=256)
aesgcm = AESGCM(key)
nonce = os.urandom(12)
ct = aesgcm.encrypt(nonce, b"sensitive data", associated_data=None)

# Or use ChaCha20-Poly1305
from cryptography.hazmat.primitives.ciphers.aead import ChaCha20Poly1305
key = ChaCha20Poly1305.generate_key()
chacha = ChaCha20Poly1305(key)
ct = chacha.encrypt(nonce, b"sensitive data", associated_data=None)
```

DETECTION AND PREVENTION:
```bash
# Scan for insecure cipher usage
pathfinder scan --project . --ruleset cpf/python/PYTHON-CRYPTO-SEC-001

# CI/CD integration
- name: Check for insecure ciphers
  run: pathfinder ci --project . --ruleset cpf/python/cryptography
```

**Code Review Checklist**:
- [ ] No usage of ARC4, RC4, Blowfish, IDEA, RC2, DES, or XOR ciphers
- [ ] All symmetric encryption uses AES-128/192/256 or ChaCha20
- [ ] Authenticated encryption modes (GCM, EAX, CCM) are used
- [ ] Key sizes meet minimum requirements (128-bit for AES, 256-bit preferred)

COMPLIANCE:
- NIST SP 800-131A: DES, 3DES (2-key), and RC4 are disallowed
- PCI DSS 3.2.1 Requirement 4.1: Strong cryptography required
- FIPS 140-2: Only AES and 3DES (3-key) approved; 3DES deprecated after 2023

REFERENCES:
- CWE-327: Use of a Broken or Risky Cryptographic Algorithm (https://cwe.mitre.org/data/definitions/327.html)
- RFC 7465: Prohibiting RC4 Cipher Suites (https://tools.ietf.org/html/rfc7465)
- Sweet32: Birthday attacks on 64-bit block ciphers (https://sweet32.info/)
- NIST SP 800-131A Rev 2: Transitioning the Use of Cryptographic Algorithms
- OWASP Cryptographic Failures (https://owasp.org/Top10/A02_2021-Cryptographic_Failures/)
"""

from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets


class CryptoCipherAlgorithms(QueryType):
    fqns = ["cryptography.hazmat.primitives.ciphers.algorithms"]


class PyCryptoCipherARC4(QueryType):
    fqns = ["Crypto.Cipher.ARC4", "Cryptodome.Cipher.ARC4"]


class PyCryptoCipherARC2(QueryType):
    fqns = ["Crypto.Cipher.ARC2", "Cryptodome.Cipher.ARC2"]


class PyCryptoCipherBlowfish(QueryType):
    fqns = ["Crypto.Cipher.Blowfish", "Cryptodome.Cipher.Blowfish"]


class PyCryptoCipherDES(QueryType):
    fqns = ["Crypto.Cipher.DES", "Cryptodome.Cipher.DES"]


class PyCryptoCipherDES3(QueryType):
    fqns = ["Crypto.Cipher.DES3", "Cryptodome.Cipher.DES3"]


class PyCryptoCipherXOR(QueryType):
    fqns = ["Crypto.Cipher.XOR", "Cryptodome.Cipher.XOR"]


@python_rule(
    id="PYTHON-CRYPTO-SEC-001",
    name="Insecure ARC4 (RC4) Cipher",
    severity="HIGH",
    category="cryptography",
    cwe="CWE-327",
    tags="python,cryptography,arc4,rc4,weak-cipher,cwe-327",
    message="ARC4/RC4 is a broken stream cipher. Use AES-GCM or ChaCha20Poly1305 instead.",
    owasp="A02:2021",
)
def detect_arc4_cipher():
    """Detects ARC4 cipher usage in cryptography and pycryptodome."""
    return CryptoCipherAlgorithms.method("ARC4")


@python_rule(
    id="PYTHON-CRYPTO-SEC-001a",
    name="Insecure ARC4 (RC4) Cipher (PyCryptodome)",
    severity="HIGH",
    category="cryptography",
    cwe="CWE-327",
    tags="python,pycryptodome,arc4,rc4,weak-cipher,cwe-327",
    message="ARC4/RC4 is a broken stream cipher. Use AES-GCM or ChaCha20Poly1305 instead.",
    owasp="A02:2021",
)
def detect_arc4_cipher_pycrypto():
    """Detects ARC4 in PyCryptodome."""
    return PyCryptoCipherARC4.method("new")


@python_rule(
    id="PYTHON-CRYPTO-SEC-002",
    name="Insecure Blowfish Cipher",
    severity="HIGH",
    category="cryptography",
    cwe="CWE-327",
    tags="python,cryptography,blowfish,weak-cipher,cwe-327",
    message="Blowfish has a 64-bit block size vulnerable to birthday attacks. Use AES instead.",
    owasp="A02:2021",
)
def detect_blowfish_cipher():
    """Detects Blowfish cipher usage in cryptography lib."""
    return CryptoCipherAlgorithms.method("Blowfish")


@python_rule(
    id="PYTHON-CRYPTO-SEC-002a",
    name="Insecure Blowfish Cipher (PyCryptodome)",
    severity="HIGH",
    category="cryptography",
    cwe="CWE-327",
    tags="python,pycryptodome,blowfish,weak-cipher,cwe-327",
    message="Blowfish has a 64-bit block size vulnerable to birthday attacks. Use AES instead.",
    owasp="A02:2021",
)
def detect_blowfish_cipher_pycrypto():
    """Detects Blowfish in PyCryptodome."""
    return PyCryptoCipherBlowfish.method("new")


@python_rule(
    id="PYTHON-CRYPTO-SEC-003",
    name="Insecure IDEA Cipher",
    severity="HIGH",
    category="cryptography",
    cwe="CWE-327",
    tags="python,cryptography,idea,weak-cipher,cwe-327",
    message="IDEA cipher is deprecated. Use AES instead.",
    owasp="A02:2021",
)
def detect_idea_cipher():
    """Detects IDEA cipher usage in cryptography lib."""
    return CryptoCipherAlgorithms.method("IDEA")


@python_rule(
    id="PYTHON-CRYPTO-SEC-004",
    name="Insecure RC2 Cipher (PyCryptodome)",
    severity="HIGH",
    category="cryptography",
    cwe="CWE-327",
    tags="python,pycryptodome,rc2,weak-cipher,cwe-327",
    message="RC2 cipher is weak. Use AES instead.",
    owasp="A02:2021",
)
def detect_rc2_cipher():
    """Detects RC2/ARC2 cipher in PyCryptodome."""
    return PyCryptoCipherARC2.method("new")


@python_rule(
    id="PYTHON-CRYPTO-SEC-005",
    name="Insecure DES Cipher",
    severity="HIGH",
    category="cryptography",
    cwe="CWE-327",
    tags="python,pycryptodome,des,weak-cipher,cwe-327",
    message="DES has a 56-bit key, easily brute-forced. Use AES instead.",
    owasp="A02:2021",
)
def detect_des_cipher():
    """Detects DES cipher in PyCryptodome."""
    return PyCryptoCipherDES.method("new")


@python_rule(
    id="PYTHON-CRYPTO-SEC-005a",
    name="Insecure Triple DES Cipher",
    severity="MEDIUM",
    category="cryptography",
    cwe="CWE-327",
    tags="python,pycryptodome,3des,triple-des,weak-cipher,cwe-327",
    message="Triple DES (3DES) is deprecated. Use AES instead.",
    owasp="A02:2021",
)
def detect_des3_cipher():
    """Detects Triple DES cipher in PyCryptodome."""
    return PyCryptoCipherDES3.method("new")


@python_rule(
    id="PYTHON-CRYPTO-SEC-006",
    name="Insecure XOR Cipher (PyCryptodome)",
    severity="HIGH",
    category="cryptography",
    cwe="CWE-327",
    tags="python,pycryptodome,xor,weak-cipher,cwe-327",
    message="XOR cipher provides no real security. Use AES instead.",
    owasp="A02:2021",
)
def detect_xor_cipher():
    """Detects XOR cipher in PyCryptodome."""
    return PyCryptoCipherXOR.method("new")
