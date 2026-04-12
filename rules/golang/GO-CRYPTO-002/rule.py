"""GO-CRYPTO-002: Use of SHA1 weak hash algorithm."""

from codepathfinder.go_rule import QueryType
from codepathfinder import flows
from codepathfinder.go_decorators import go_rule


class GoCryptoSHA1(QueryType):
    fqns = ["crypto/sha1"]
    patterns = ["sha1.*"]
    match_subclasses = False


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
    """Detect use of SHA1 hashing (crypto/sha1.New or sha1.Sum)."""
    return GoCryptoSHA1.method("New", "Sum")
