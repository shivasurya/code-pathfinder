"""GO-JWT-002: JWT token parsed without signature verification (ParseUnverified)."""

from codepathfinder.go_rule import QueryType
from codepathfinder import flows
from rules.go_decorators import go_rule


class GoJWTLib(QueryType):
    fqns = ["github.com/golang-jwt/jwt/v5", "github.com/golang-jwt/jwt", "github.com/dgrijalva/jwt-go"]
    patterns = ["jwt.*"]
    match_subclasses = False


@go_rule(
    id="GO-JWT-002",
    severity="HIGH",
    cwe="CWE-345",
    owasp="A08:2021",
    tags="go,security,jwt,parse-unverified,integrity,CWE-345,OWASP-A08",
    message=(
        "Detected use of jwt.ParseUnverified() which skips signature verification. "
        "Tokens parsed with ParseUnverified are not validated — an attacker can forge "
        "arbitrary claims by crafting a JWT without a valid signature. "
        "Use jwt.Parse() with a key function that validates the signing algorithm and secret."
    ),
)
def detect_jwt_parse_unverified():
    """Detect use of ParseUnverified() that skips JWT signature validation."""
    return GoJWTLib.method("ParseUnverified")
