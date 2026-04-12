"""GO-CRYPTO-004: Use of RC4 cipher (broken algorithm)."""

from codepathfinder.go_rule import QueryType
from codepathfinder import flows
from rules.go_decorators import go_rule


class GoCryptoRC4(QueryType):
    fqns = ["crypto/rc4"]
    patterns = ["rc4.*"]
    match_subclasses = False


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
    """Detect use of RC4 cipher (crypto/rc4)."""
    return GoCryptoRC4.method("NewCipher")
