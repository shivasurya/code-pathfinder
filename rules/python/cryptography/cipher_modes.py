"""
Python Cipher Mode Security Rules

PYTHON-CRYPTO-SEC-030: ECB Mode Usage (cryptography lib)
PYTHON-CRYPTO-SEC-031: Unauthenticated Cipher Mode (cryptography lib)
PYTHON-CRYPTO-SEC-032: Unauthenticated Cipher Mode (PyCryptodome)

Security Impact: MEDIUM to HIGH
CWE: CWE-327 (Use of a Broken or Risky Cryptographic Algorithm)
OWASP: A02:2021 - Cryptographic Failures

DESCRIPTION:
These rules detect the use of insecure or unauthenticated cipher modes in Python
cryptographic libraries. Cipher mode selection is critical to the security of symmetric
encryption: the wrong mode can leak plaintext patterns, enable ciphertext manipulation,
or allow padding oracle attacks even when using a strong cipher like AES.

Detected insecure modes:
- **ECB (Electronic Codebook)**: Encrypts each block independently; identical plaintext blocks
  produce identical ciphertext blocks, leaking data patterns (the "ECB penguin" problem)
- **CBC, CTR, CFB, OFB without authentication**: Provide confidentiality but not integrity;
  ciphertext can be tampered with undetected (bit-flipping attacks, padding oracle attacks)

SECURITY IMPLICATIONS:

**1. Pattern Leakage (ECB)**:
ECB mode preserves plaintext patterns in ciphertext. Any repeated data block produces
identical ciphertext, allowing attackers to identify structure, detect duplicates, and
infer content without decrypting.

**2. Ciphertext Manipulation (Unauthenticated Modes)**:
Without authenticated encryption, attackers can modify ciphertext and the recipient
cannot detect tampering. In CBC mode, this enables padding oracle attacks (e.g.,
POODLE, Lucky13). In CTR mode, bit-flipping directly modifies the corresponding
plaintext bits.

**3. Padding Oracle Attacks**:
CBC mode with PKCS#7 padding is vulnerable to padding oracle attacks where an attacker
can decrypt entire messages by observing error responses from the decryption endpoint.

VULNERABLE EXAMPLE:
```python
from cryptography.hazmat.primitives.ciphers import Cipher, algorithms, modes

# VULNERABLE: ECB mode leaks plaintext patterns
cipher = Cipher(algorithms.AES(key), modes.ECB())
encryptor = cipher.encryptor()
ct = encryptor.update(plaintext) + encryptor.finalize()

# VULNERABLE: CBC without HMAC (no integrity protection)
cipher = Cipher(algorithms.AES(key), modes.CBC(iv))
encryptor = cipher.encryptor()
ct = encryptor.update(plaintext) + encryptor.finalize()
# Attacker can flip bits in ciphertext without detection!

# VULNERABLE: PyCryptodome AES in ECB mode
from Crypto.Cipher import AES
cipher = AES.new(key, AES.MODE_ECB)
ct = cipher.encrypt(plaintext)
```

SECURE EXAMPLE:
```python
from cryptography.hazmat.primitives.ciphers.aead import AESGCM
import os

# SECURE: AES-GCM provides both confidentiality AND integrity
key = AESGCM.generate_key(bit_length=256)
aesgcm = AESGCM(key)
nonce = os.urandom(12)
ct = aesgcm.encrypt(nonce, plaintext, associated_data=b"header")
# Tampered ciphertext will raise InvalidTag on decrypt

# SECURE: PyCryptodome AES-GCM
from Crypto.Cipher import AES
cipher = AES.new(key, AES.MODE_GCM)
ct, tag = cipher.encrypt_and_digest(plaintext)

# SECURE: If CBC is required, add HMAC (Encrypt-then-MAC)
import hmac, hashlib
cipher = Cipher(algorithms.AES(key_enc), modes.CBC(iv))
ct = cipher.encryptor().update(plaintext) + cipher.encryptor().finalize()
mac = hmac.new(key_mac, ct, hashlib.sha256).digest()
# Verify MAC before decrypting
```

DETECTION AND PREVENTION:
```bash
# Scan for insecure cipher modes
pathfinder scan --project . --ruleset cpf/python/PYTHON-CRYPTO-SEC-030

# CI/CD integration
- name: Check cipher mode security
  run: pathfinder ci --project . --ruleset cpf/python/cryptography
```

**Code Review Checklist**:
- [ ] No ECB mode usage for any data larger than a single block
- [ ] Authenticated encryption (GCM, EAX, CCM, SIV) is preferred
- [ ] If CBC/CTR is used, HMAC is applied using Encrypt-then-MAC pattern
- [ ] Nonces/IVs are randomly generated and never reused with the same key
- [ ] PyCryptodome uses MODE_GCM or MODE_EAX rather than MODE_ECB or MODE_CBC

COMPLIANCE:
- NIST SP 800-38A: Defines approved modes of operation (ECB discouraged for multi-block data)
- NIST SP 800-38D: GCM mode specification and recommendations
- PCI DSS: Requires strong cryptography with authenticated encryption
- FIPS 140-2: Approved modes include CBC, CTR, GCM; ECB only for single blocks

REFERENCES:
- CWE-327: Use of a Broken or Risky Cryptographic Algorithm (https://cwe.mitre.org/data/definitions/327.html)
- NIST SP 800-38A: Recommendation for Block Cipher Modes of Operation
- The ECB Penguin: Why ECB mode is insecure (https://blog.filippo.io/the-ecb-penguin/)
- Padding Oracle Attacks: Vaudenay (2002), POODLE (CVE-2014-3566)
- OWASP Cryptographic Failures (https://owasp.org/Top10/A02_2021-Cryptographic_Failures/)
"""

from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets


class CryptoModes(QueryType):
    fqns = ["cryptography.hazmat.primitives.ciphers.modes"]


class PyCryptoAES(QueryType):
    fqns = ["Crypto.Cipher.AES", "Cryptodome.Cipher.AES"]


@python_rule(
    id="PYTHON-CRYPTO-SEC-030",
    name="ECB Mode Usage",
    severity="HIGH",
    category="cryptography",
    cwe="CWE-327",
    tags="python,cryptography,ecb,cipher-mode,weak-mode,cwe-327",
    message="ECB mode does not provide semantic security. Use CBC, CTR, or GCM instead.",
    owasp="A02:2021",
)
def detect_ecb_mode():
    """Detects ECB cipher mode in cryptography lib."""
    return CryptoModes.method("ECB")


@python_rule(
    id="PYTHON-CRYPTO-SEC-031",
    name="Unauthenticated Cipher Mode (cryptography)",
    severity="MEDIUM",
    category="cryptography",
    cwe="CWE-327",
    tags="python,cryptography,cipher-mode,unauthenticated,cwe-327",
    message="CBC/CTR/CFB/OFB without authentication (HMAC). Use GCM or add HMAC.",
    owasp="A02:2021",
)
def detect_unauthenticated_mode_crypto():
    """Audit: detects CBC/CTR/CFB/OFB mode usage that may lack authentication."""
    return CryptoModes.method("CBC", "CTR", "CFB", "OFB")


@python_rule(
    id="PYTHON-CRYPTO-SEC-032",
    name="Unauthenticated Cipher Mode (PyCryptodome)",
    severity="MEDIUM",
    category="cryptography",
    cwe="CWE-327",
    tags="python,pycryptodome,cipher-mode,unauthenticated,cwe-327",
    message="AES in non-GCM/EAX/SIV mode may lack authentication. Use MODE_GCM or MODE_EAX.",
    owasp="A02:2021",
)
def detect_aes_pycrypto_audit():
    """Audit: detects AES.new() usage in PyCryptodome for mode review."""
    return PyCryptoAES.method("new")
