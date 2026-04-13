"""
GO-JWT Rules: JSON Web Token security vulnerabilities.

GO-JWT-001: JWT signed with the 'none' algorithm
GO-JWT-002: JWT token parsed without signature verification (ParseUnverified)

Security Impact: CRITICAL
CWE: CWE-327 (Use of a Broken or Risky Cryptographic Algorithm)
     CWE-345 (Insufficient Verification of Data Authenticity)
OWASP: A02:2021 — Cryptographic Failures
       A08:2021 — Software and Data Integrity Failures

DESCRIPTION:
JWT tokens with the 'none' algorithm have no cryptographic signature — any
attacker can forge arbitrary claims (e.g., admin=true, user_id=any_value).
This bypasses all authentication and authorization.

ParseUnverified() skips signature verification entirely. If a JWT from an
untrusted source is parsed without verification, an attacker can forge tokens
with arbitrary claims that the application will trust.

VULNERABLE EXAMPLES:
    // JWT none algorithm — tokens can be forged by anyone
    token := jwt.New(jwt.SigningMethodNone)
    signedToken, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)

    // ParseUnverified — signature not checked
    token, _, _ := parser.ParseUnverified(tokenString, &Claims{})

SECURE EXAMPLES:
    // Always use a strong algorithm with a secret key
    token := jwt.New(jwt.SigningMethodHS256)
    signedToken, _ := token.SignedString([]byte(os.Getenv("JWT_SECRET")))

    // Always verify the signature on parsing
    token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
        if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
        }
        return []byte(os.Getenv("JWT_SECRET")), nil
    })

REFERENCES:
- CWE-327: https://cwe.mitre.org/data/definitions/327.html
- CWE-345: https://cwe.mitre.org/data/definitions/345.html
- JWT Security: https://auth0.com/blog/critical-vulnerabilities-in-json-web-token-libraries/
- RFC 8725 (JWT Best Current Practices): https://www.rfc-editor.org/rfc/rfc8725
"""

from codepathfinder.go_rule import QueryType
from codepathfinder import calls
from codepathfinder.go_decorators import go_rule


class GoJWTLib(QueryType):
    """github.com/golang-jwt/jwt — JWT library for Go."""

    fqns = ["github.com/golang-jwt/jwt/v5", "github.com/golang-jwt/jwt", "github.com/dgrijalva/jwt-go"]
    patterns = ["jwt.*"]
    match_subclasses = False


@go_rule(
    id="GO-JWT-001",
    severity="CRITICAL",
    cwe="CWE-327",
    owasp="A02:2021",
    tags="go,security,jwt,none-alg,broken-auth,CWE-327,OWASP-A02",
    message=(
        "Detected use of the JWT 'none' signing algorithm. "
        "jwt.SigningMethodNone / jwt.UnsafeAllowNoneSignatureType creates tokens "
        "with no cryptographic signature — any attacker can forge arbitrary JWT claims "
        "and bypass authentication entirely. "
        "Use jwt.SigningMethodHS256 or jwt.SigningMethodRS256 with a strong secret."
    ),
)
def detect_jwt_none_algorithm():
    """Detect use of JWT none algorithm (SigningMethodNone / UnsafeAllowNoneSignatureType).

    The 'none' algorithm disables JWT signature verification. Any token with
    algorithm=none and any claims will be accepted as valid.

    Bad:  jwt.New(jwt.SigningMethodNone)
          token.SignedString(jwt.UnsafeAllowNoneSignatureType)
    Good: jwt.New(jwt.SigningMethodHS256)
          token.SignedString([]byte(secret))
    """
    # SigningMethodNone and UnsafeAllowNoneSignatureType are package-level variables,
    # accessed as attribute reads (jwt.SigningMethodNone), not method calls.
    return GoJWTLib.attr(
        "SigningMethodNone",
        "UnsafeAllowNoneSignatureType",
    )


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
    """Detect use of ParseUnverified() that skips JWT signature validation.

    ParseUnverified() is only safe when the signature has already been verified
    by an upstream system. Using it on untrusted input allows attackers to forge
    any JWT claims they want.

    Bad:  parser.ParseUnverified(tokenString, &Claims{})
    Good: jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) { ... })
    """
    return GoJWTLib.method("ParseUnverified")
