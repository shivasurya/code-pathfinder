"""GO-CRYPTO-001: Use of MD5 weak hash algorithm."""

from codepathfinder.go_rule import QueryType
from codepathfinder import flows
from codepathfinder.go_decorators import go_rule


class GoCryptoMD5(QueryType):
    fqns = ["crypto/md5"]
    patterns = ["md5.*"]
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
    """Detect use of MD5 hashing (crypto/md5.New or md5.Sum)."""
    return GoCryptoMD5.method("New", "Sum")
