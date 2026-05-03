"""
GO-CRYPTO Rules: Weak and broken cryptographic algorithms.

GO-CRYPTO-001: Use of MD5 hash algorithm (weak hash)
GO-CRYPTO-002: Use of SHA1 hash algorithm (weak hash)
GO-CRYPTO-003: Use of DES cipher (broken algorithm)
GO-CRYPTO-004: Use of RC4 cipher (broken algorithm)
GO-CRYPTO-005: MD5 used for password hashing (critical misuse)

Security Impact: HIGH
CWE: CWE-327 (Use of a Broken or Risky Cryptographic Algorithm)
     CWE-328 (Use of Weak Hash)
OWASP: A02:2021 — Cryptographic Failures

DESCRIPTION:
MD5 and SHA1 are cryptographically broken — both have known collision attacks.
DES uses a 56-bit key, brute-forceable in hours. RC4 has multiple biases that
make ciphertexts predictable. None of these should be used for security-sensitive
operations. Use SHA256/SHA512 for hashing, AES-GCM for symmetric encryption.

Using MD5 specifically for password storage is a critical vulnerability — MD5
hashes can be cracked with rainbow tables in seconds. Use bcrypt, scrypt, or
argon2 for password hashing.

VULNERABLE EXAMPLES:
    import "crypto/md5"
    h := md5.New()                    // VULNERABLE: MD5 is collision-broken
    password_hash := md5.Sum([]byte(password))  // CRITICAL: MD5 for passwords

    import "crypto/des"
    cipher, _ := des.NewCipher(key)  // VULNERABLE: 56-bit key, brute-forceable

SECURE EXAMPLES:
    import "crypto/sha256"
    h := sha256.New()                 // SAFE: collision-resistant

    import "golang.org/x/crypto/bcrypt"
    hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

REFERENCES:
- CWE-327: https://cwe.mitre.org/data/definitions/327.html
- CWE-328: https://cwe.mitre.org/data/definitions/328.html
- OWASP Cryptographic Failures: https://owasp.org/Top10/A02_2021-Cryptographic_Failures
"""

from codepathfinder.go_rule import (
    GoHTTPRequest,
    GoGinContext,
    GoEchoContext,
    QueryType,
)
from codepathfinder import calls, flows
from codepathfinder.presets import PropagationPresets
from codepathfinder.go_decorators import go_rule


class GoCryptoMD5(QueryType):
    """crypto/md5 — weak MD5 hash algorithm."""

    fqns = ["crypto/md5"]
    patterns = ["md5.*"]
    match_subclasses = False


class GoCryptoSHA1(QueryType):
    """crypto/sha1 — weak SHA1 hash algorithm."""

    fqns = ["crypto/sha1"]
    patterns = ["sha1.*"]
    match_subclasses = False


class GoCryptoDES(QueryType):
    """crypto/des — broken DES/3DES cipher."""

    fqns = ["crypto/des"]
    patterns = ["des.*"]
    match_subclasses = False


class GoCryptoRC4(QueryType):
    """crypto/rc4 — broken RC4 stream cipher."""

    fqns = ["crypto/rc4"]
    patterns = ["rc4.*"]
    match_subclasses = False


@go_rule(
    id="GO-CRYPTO-001",
    severity="HIGH",
    cwe="CWE-328",
    owasp="A02:2021",
    tags="go,security,crypto,md5,weak-hash,CWE-328,OWASP-A02",
    message=(
        "Detected use of the MD5 hash algorithm (crypto/md5). "
        "MD5 is cryptographically broken — it has known collision attacks and "
        "should not be used for any security-sensitive purpose. "
        "Use crypto/sha256 or crypto/sha512 instead."
    ),
)
def detect_md5_weak_hash():
    """Detect use of MD5 hashing (crypto/md5.New or md5.Sum).

    MD5 is collision-broken. Attackers can forge arbitrary data with the same
    MD5 hash. Do not use MD5 for digital signatures, integrity checks, or
    any security-relevant hashing.

    Bad:  h := md5.New(); md5.Sum(data)
    Good: h := sha256.New(); sha256.Sum256(data)
    """
    return GoCryptoMD5.method("New", "Sum")


@go_rule(
    id="GO-CRYPTO-002",
    severity="HIGH",
    cwe="CWE-328",
    owasp="A02:2021",
    tags="go,security,crypto,sha1,weak-hash,CWE-328,OWASP-A02",
    message=(
        "Detected use of the SHA1 hash algorithm (crypto/sha1). "
        "SHA1 is cryptographically weak — it has known collision attacks (SHAttered, 2017). "
        "Use crypto/sha256 or crypto/sha512 instead for new code."
    ),
)
def detect_sha1_weak_hash():
    """Detect use of SHA1 hashing (crypto/sha1.New or sha1.Sum).

    SHA1 is considered weak for cryptographic use since the SHAttered attack.
    Certificate authorities no longer issue SHA1 certificates.

    Bad:  h := sha1.New(); sha1.Sum(data)
    Good: h := sha256.New(); sha256.Sum256(data)
    """
    return GoCryptoSHA1.method("New", "Sum")


@go_rule(
    id="GO-CRYPTO-003",
    severity="HIGH",
    cwe="CWE-327",
    owasp="A02:2021",
    tags="go,security,crypto,des,broken-cipher,CWE-327,OWASP-A02",
    message=(
        "Detected use of the DES/3DES cipher (crypto/des). "
        "DES uses a 56-bit key that can be brute-forced in hours with modern hardware. "
        "3DES (Triple DES) is deprecated by NIST (SP 800-131A). "
        "Use AES-GCM (crypto/aes + crypto/cipher) instead."
    ),
)
def detect_des_cipher():
    """Detect use of DES or Triple DES cipher (crypto/des).

    DES has an effective key length of 56 bits — vulnerable to exhaustive search.
    3DES (TripleDES) is officially deprecated. Both should be replaced with AES.

    Bad:  des.NewCipher(key); des.NewTripleDESCipher(key)
    Good: aes.NewCipher(key) with GCM mode
    """
    return GoCryptoDES.method("NewCipher", "NewTripleDESCipher")


@go_rule(
    id="GO-CRYPTO-004",
    severity="HIGH",
    cwe="CWE-327",
    owasp="A02:2021",
    tags="go,security,crypto,rc4,broken-cipher,CWE-327,OWASP-A02",
    message=(
        "Detected use of the RC4 stream cipher (crypto/rc4). "
        "RC4 has multiple statistical biases that allow plaintext recovery. "
        "It is prohibited in TLS (RFC 7465) and should not be used for any purpose. "
        "Use AES-GCM (crypto/aes + crypto/cipher) instead."
    ),
)
def detect_rc4_cipher():
    """Detect use of RC4 cipher (crypto/rc4).

    RC4 is banned in TLS (RFC 7465) and FIPS. Its keystream has known biases
    that attackers can exploit to recover plaintext.

    Bad:  rc4.NewCipher(key)
    Good: aes.NewCipher(key) with cipher.NewGCM
    """
    return GoCryptoRC4.method("NewCipher")


@go_rule(
    id="GO-CRYPTO-005",
    severity="CRITICAL",
    cwe="CWE-327",
    owasp="A02:2021",
    tags="go,security,crypto,md5,password-hash,CWE-327,OWASP-A02",
    message=(
        "MD5 is being used to hash passwords. MD5 is completely unsuitable for "
        "password storage — MD5 hashes can be cracked in seconds using GPU-accelerated "
        "rainbow tables or dictionary attacks. "
        "Use bcrypt (golang.org/x/crypto/bcrypt), scrypt, or argon2 for password hashing."
    ),
)
def detect_md5_password_hash():
    """Detect MD5 used for password hashing.

    Using MD5 for password storage is a critical security failure. MD5 can be
    cracked at billions of hashes per second on commodity hardware.

    Bad:  md5.Sum([]byte(password))
    Good: bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    """
    return flows(
        from_sources=[
            GoCryptoMD5.method("New", "Sum"),
        ],
        to_sinks=[
            calls("*password*", "hashPassword", "storePassword", "savePassword"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
