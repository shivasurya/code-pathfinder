"""GO-CRYPTO-005: MD5 used for password hashing (critical misuse)."""

from codepathfinder.go_rule import QueryType
from codepathfinder import calls, flows
from codepathfinder.presets import PropagationPresets
from rules.go_decorators import go_rule


class GoCryptoMD5(QueryType):
    fqns = ["crypto/md5"]
    patterns = ["md5.*"]
    match_subclasses = False


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
    """Detect MD5 used for password hashing."""
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
