"""GO-CRYPTO-003: Use of DES/3DES cipher (broken algorithm)."""

from codepathfinder.go_rule import QueryType
from codepathfinder import flows
from codepathfinder.go_decorators import go_rule


class GoCryptoDES(QueryType):
    fqns = ["crypto/des"]
    patterns = ["des.*"]
    match_subclasses = False


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
    """Detect use of DES or Triple DES cipher (crypto/des)."""
    return GoCryptoDES.method("NewCipher", "NewTripleDESCipher")
