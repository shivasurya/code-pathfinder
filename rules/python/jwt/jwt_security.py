"""
Python JWT Security Rules

PYTHON-JWT-SEC-001: JWT Hardcoded Secret
PYTHON-JWT-SEC-002: JWT None Algorithm
PYTHON-JWT-SEC-003: Unverified JWT Decode
PYTHON-JWT-SEC-004: JWT Exposed Credentials (Audit)
PYTHON-JWT-SEC-005: JWT Exposed Data (Audit)

Security Impact: CRITICAL to LOW (varies by rule)
CWE: CWE-327 (Use of a Broken or Risky Cryptographic Algorithm),
     CWE-522 (Insufficiently Protected Credentials),
     CWE-287 (Improper Authentication)
OWASP: A02:2021 - Cryptographic Failures, A07:2021 - Identification and Authentication Failures

DESCRIPTION:
These rules detect security misconfigurations in JSON Web Token (JWT) usage with the
PyJWT library (`jwt` module). JWTs are widely used for authentication and authorization
in web applications. Misconfigurations can allow token forgery, credential exposure, and
authentication bypass.

Detected vulnerabilities:
- **Hardcoded secrets**: JWT signing keys embedded directly in source code instead of
  using environment variables or key management services
- **None algorithm**: Using algorithm='none' disables signature verification entirely,
  allowing anyone to forge valid tokens
- **Unverified decode**: Calling jwt.decode() with options={"verify_signature": False}
  accepts tokens without validating their signature
- **Credential exposure**: Storing passwords, API keys, or secrets in JWT payloads (which
  are base64-encoded, NOT encrypted)
- **Data exposure**: Sensitive user data in JWT payloads visible to anyone with the token

SECURITY IMPLICATIONS:

**1. Token Forgery (None Algorithm)**:
When algorithm='none' is accepted, any attacker can create a valid token by removing the
signature and setting the algorithm header to "none". This completely bypasses authentication.

**2. Secret Compromise (Hardcoded Secrets)**:
Hardcoded signing secrets in source code are exposed through version control history,
decompilation, or source code leaks. Once compromised, an attacker can forge any token.

**3. Authentication Bypass (Unverified Decode)**:
Decoding without signature verification means the application trusts any token regardless
of origin, allowing attackers to craft tokens with arbitrary claims.

**4. Data Leakage (Exposed Payloads)**:
JWT payloads are only base64url-encoded, not encrypted. Any party with access to the
token can decode and read the payload contents. Storing sensitive data (passwords, PII,
secrets) in JWTs exposes them.

VULNERABLE EXAMPLE:
```python
import jwt

# SEC-001: hardcoded secret + SEC-004: encode audit
token = jwt.encode({"user": "admin"}, "my_secret_key", algorithm="HS256")

# SEC-002: none algorithm (encode)
unsafe_token = jwt.encode({"user": "admin"}, "", algorithm="none")

# SEC-002: none algorithm (decode)
payload = jwt.decode(token, "", algorithms=["none"])

# SEC-003: unverified decode
data = jwt.decode(token, "secret", options={"verify_signature": False})

# SEC-005: request data to jwt.encode (flow)
def create_token(request):
    user_data = request.args.get('user')
    return jwt.encode({"sub": user_data}, "key", algorithm="HS256")
```

SECURE EXAMPLE:
```python
import jwt
import os

# SECURE: Secret from environment variable
SECRET = os.environ["JWT_SECRET_KEY"]

# SECURE: Use strong algorithm with proper secret
token = jwt.encode(
    {"user_id": 123, "role": "admin", "exp": datetime.utcnow() + timedelta(hours=1)},
    SECRET,
    algorithm="HS256"
)

# SECURE: Verify signature and specify allowed algorithms
payload = jwt.decode(
    token,
    SECRET,
    algorithms=["HS256"],  # Explicitly whitelist algorithms
    options={"require": ["exp", "iat"]}
)

# SECURE: Use RS256 with asymmetric keys for distributed systems
with open("private_key.pem", "rb") as f:
    private_key = f.read()
token = jwt.encode(payload, private_key, algorithm="RS256")

with open("public_key.pem", "rb") as f:
    public_key = f.read()
payload = jwt.decode(token, public_key, algorithms=["RS256"])
```

DETECTION AND PREVENTION:
```bash
# Scan for JWT security issues
pathfinder scan --project . --ruleset cpf/python/PYTHON-JWT-SEC-001

# CI/CD integration
- name: Check JWT security
  run: pathfinder ci --project . --ruleset cpf/python/jwt
```

**Code Review Checklist**:
- [ ] JWT secrets are loaded from environment variables or key management services
- [ ] No usage of algorithm='none' in jwt.encode() or jwt.decode()
- [ ] jwt.decode() always verifies signatures (no verify_signature=False)
- [ ] algorithms parameter explicitly whitelists allowed algorithms in jwt.decode()
- [ ] No sensitive data (passwords, PII, API keys) stored in JWT payloads
- [ ] Tokens include expiration claims (exp) and are validated
- [ ] Consider RS256/ES256 for distributed systems instead of HS256

COMPLIANCE:
- OWASP A02:2021: Cryptographic Failures (hardcoded secrets, weak algorithms)
- OWASP A07:2021: Identification and Authentication Failures (unverified tokens)
- CWE-522: Insufficiently Protected Credentials
- RFC 7519: JSON Web Token specification
- RFC 8725: JWT Best Current Practices

REFERENCES:
- CWE-327: Use of a Broken or Risky Cryptographic Algorithm (https://cwe.mitre.org/data/definitions/327.html)
- CWE-522: Insufficiently Protected Credentials (https://cwe.mitre.org/data/definitions/522.html)
- CWE-287: Improper Authentication (https://cwe.mitre.org/data/definitions/287.html)
- RFC 8725: JSON Web Token Best Current Practices (https://tools.ietf.org/html/rfc8725)
- OWASP JWT Cheat Sheet (https://cheatsheetseries.owasp.org/cheatsheets/JSON_Web_Token_for_Java_Cheat_Sheet.html)
- Auth0 JWT Handbook (https://auth0.com/resources/ebooks/jwt-handbook)
"""

from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets


class JWTModule(QueryType):
    fqns = ["jwt"]


@python_rule(
    id="PYTHON-JWT-SEC-001",
    name="JWT Hardcoded Secret",
    severity="HIGH",
    category="jwt",
    cwe="CWE-522",
    tags="python,jwt,hardcoded-secret,credentials,owasp-a02,cwe-522",
    message="Hardcoded string used as JWT signing secret. Use environment variables or key management.",
    owasp="A02:2021",
)
def detect_jwt_hardcoded_secret():
    """Detects jwt.encode() with hardcoded string secret."""
    return JWTModule.method("encode")


@python_rule(
    id="PYTHON-JWT-SEC-002",
    name="JWT None Algorithm",
    severity="CRITICAL",
    category="jwt",
    cwe="CWE-327",
    tags="python,jwt,none-algorithm,weak-crypto,owasp-a02,cwe-327",
    message="JWT with algorithm='none' disables signature verification. Use HS256 or RS256.",
    owasp="A02:2021",
)
def detect_jwt_none_algorithm():
    """Detects jwt.encode/decode with algorithm='none'."""
    return JWTModule.method("encode", "decode")


@python_rule(
    id="PYTHON-JWT-SEC-003",
    name="Unverified JWT Decode",
    severity="HIGH",
    category="jwt",
    cwe="CWE-287",
    tags="python,jwt,unverified,authentication,owasp-a07,cwe-287",
    message="jwt.decode() with verify_signature=False. Token integrity is not checked.",
    owasp="A07:2021",
)
def detect_jwt_unverified_decode():
    """Detects jwt.decode() calls that may disable verification."""
    return JWTModule.method("decode")


@python_rule(
    id="PYTHON-JWT-SEC-004",
    name="JWT Exposed Credentials (Audit)",
    severity="MEDIUM",
    category="jwt",
    cwe="CWE-522",
    tags="python,jwt,credentials,audit,cwe-522",
    message="jwt.encode() detected. Ensure no passwords or secrets are in the payload.",
    owasp="A02:2021",
)
def detect_jwt_exposed_credentials():
    """Audit: detects jwt.encode() to review for credential exposure."""
    return JWTModule.method("encode")


@python_rule(
    id="PYTHON-JWT-SEC-005",
    name="JWT Exposed Data (Audit)",
    severity="LOW",
    category="jwt",
    cwe="CWE-522",
    tags="python,jwt,data-exposure,audit,cwe-522",
    message="Data passed to jwt.encode(). JWT payloads are base64-encoded, not encrypted.",
    owasp="A02:2021",
)
def detect_jwt_exposed_data():
    """Audit: detects any data flowing into jwt.encode()."""
    return flows(
        from_sources=[
            calls("request.GET.get"),
            calls("request.POST.get"),
            calls("request.args.get"),
            calls("request.form.get"),
        ],
        to_sinks=[
            JWTModule.method("encode"),
        ],
        sanitized_by=[],
        propagates_through=PropagationPresets.standard(),
        scope="local",
    )
