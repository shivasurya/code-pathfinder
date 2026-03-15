"""
SSL/TLS Configuration Security Rules for Python

Rules in this file:
- PYTHON-LANG-SEC-050: Unverified SSL Context (CWE-295)
- PYTHON-LANG-SEC-051: Weak SSL/TLS Protocol Version (CWE-326)
- PYTHON-LANG-SEC-052: Deprecated ssl.wrap_socket() (CWE-326)
- PYTHON-LANG-SEC-053: Disabled Certificate Validation (CWE-295)
- PYTHON-LANG-SEC-054: Insecure HTTP Connection (CWE-319)

Security Impact: HIGH
CWE: CWE-295 (Improper Certificate Validation)
OWASP: A07:2021 - Identification and Authentication Failures

DESCRIPTION:
Proper SSL/TLS configuration is essential for establishing secure encrypted
communication channels. Python's ssl module provides fine-grained control over
TLS protocol versions, certificate verification, and cipher suites. Misconfigurations
such as disabling certificate verification, using deprecated protocol versions
(SSLv2, SSLv3, TLS 1.0, TLS 1.1), or using the deprecated ssl.wrap_socket()
function undermine transport security and expose applications to interception.

SECURITY IMPLICATIONS:
Disabling SSL certificate verification (ssl._create_unverified_context() or
verify=False in requests) removes the primary defense against man-in-the-middle
(MITM) attacks. An attacker who controls any network hop between client and
server can present a fraudulent certificate, intercept all encrypted traffic,
and modify data in transit. Using weak TLS protocol versions (SSLv2, SSLv3,
TLS 1.0, TLS 1.1) exposes connections to known protocol-level attacks including
POODLE, BEAST, CRIME, and DROWN. The deprecated ssl.wrap_socket() function
lacks modern security defaults and does not perform hostname verification by
default.

    # Attack scenario: MITM with disabled cert verification
    # Attacker performs ARP spoofing on corporate WiFi
    response = requests.get("https://bank.com/api/transfer",
                           verify=False)  # Attacker intercepts, modifies transfer amount

VULNERABLE EXAMPLE:
```python
import ssl, requests
# Disabled certificate verification
ctx = ssl._create_unverified_context()
response = requests.get("https://api.example.com", verify=False)
# Weak TLS version allowing protocol attacks
ctx = ssl.SSLContext(ssl.PROTOCOL_TLSv1)
# Deprecated wrap_socket without hostname checking
ssl.wrap_socket(sock, certfile="cert.pem")
# Plaintext HTTP connection
conn = http.client.HTTPConnection("api.example.com")
```

SECURE EXAMPLE:
```python
import ssl, requests
# Use default context with proper certificate verification
ctx = ssl.create_default_context()
response = requests.get("https://api.example.com")  # verify=True is default
# Enforce minimum TLS 1.2
ctx = ssl.SSLContext(ssl.PROTOCOL_TLS_CLIENT)
ctx.minimum_version = ssl.TLSVersion.TLSv1_2
# Use SSLContext.wrap_socket with hostname checking
ctx = ssl.create_default_context()
secure_sock = ctx.wrap_socket(sock, server_hostname="api.example.com")
# Always use HTTPS
conn = http.client.HTTPSConnection("api.example.com")
```

DETECTION AND PREVENTION:
- Use ssl.create_default_context() which enables certificate verification by default
- Set minimum TLS version to 1.2 via ctx.minimum_version = ssl.TLSVersion.TLSv1_2
- Never pass verify=False to requests library calls in production code
- Replace ssl.wrap_socket() with SSLContext.wrap_socket() for modern security defaults
- Use HTTPSConnection instead of HTTPConnection for all API communication
- Pin certificates or use certificate transparency monitoring for critical services

COMPLIANCE:
- CWE-295: Improper Certificate Validation
- CWE-326: Inadequate Encryption Strength
- CWE-319: Cleartext Transmission of Sensitive Information
- OWASP A02:2021 - Cryptographic Failures
- OWASP A07:2021 - Identification and Authentication Failures
- PCI DSS v4.0 Requirement 4.2: Strong cryptography for data transmission
- NIST SP 800-52 Rev 2: Guidelines for TLS Implementations
- RFC 8996: Deprecating TLS 1.0 and TLS 1.1

REFERENCES:
- https://cwe.mitre.org/data/definitions/295.html
- https://cwe.mitre.org/data/definitions/326.html
- https://owasp.org/Top10/A02_2021-Cryptographic_Failures/
- https://docs.python.org/3/library/ssl.html#ssl.create_default_context
- https://docs.python.org/3/library/ssl.html#ssl-security
- https://requests.readthedocs.io/en/latest/user/advanced/#ssl-cert-verification
"""

from rules.python_decorators import python_rule
from codepathfinder import calls, QueryType


class SSLModule(QueryType):
    fqns = ["ssl"]


class RequestsLib(QueryType):
    fqns = ["requests"]


class HttplibModule(QueryType):
    fqns = ["http.client"]


@python_rule(
    id="PYTHON-LANG-SEC-050",
    name="Unverified SSL Context",
    severity="HIGH",
    category="lang",
    cwe="CWE-295",
    tags="python,ssl,unverified-context,certificate,owasp-a07,cwe-295",
    message="ssl._create_unverified_context() disables certificate verification. Use ssl.create_default_context().",
    owasp="A07:2021",
)
def detect_unverified_ssl():
    """Detects ssl._create_unverified_context() usage."""
    return SSLModule.method("_create_unverified_context")


@python_rule(
    id="PYTHON-LANG-SEC-051",
    name="Weak SSL/TLS Protocol Version",
    severity="HIGH",
    category="lang",
    cwe="CWE-326",
    tags="python,ssl,weak-tls,protocol-version,cwe-326",
    message="Weak SSL/TLS version detected (SSLv2/3 or TLSv1/1.1). Use TLS 1.2+ minimum.",
    owasp="A02:2021",
)
def detect_weak_ssl():
    """Detects SSLContext with weak protocol versions."""
    return SSLModule.method("SSLContext").where(0, "ssl.PROTOCOL_SSLv2")


@python_rule(
    id="PYTHON-LANG-SEC-052",
    name="Deprecated ssl.wrap_socket()",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-326",
    tags="python,ssl,wrap-socket,deprecated,cwe-326",
    message="ssl.wrap_socket() is deprecated since Python 3.7. Use SSLContext.wrap_socket() instead.",
    owasp="A02:2021",
)
def detect_wrap_socket():
    """Detects deprecated ssl.wrap_socket() usage."""
    return SSLModule.method("wrap_socket")


@python_rule(
    id="PYTHON-LANG-SEC-053",
    name="Disabled Certificate Validation",
    severity="HIGH",
    category="lang",
    cwe="CWE-295",
    tags="python,ssl,cert-validation,mitm,cwe-295",
    message="Certificate validation disabled (verify=False or CERT_NONE). Enable certificate verification.",
    owasp="A07:2021",
)
def detect_disabled_cert():
    """Detects requests.get(verify=False) and similar patterns."""
    return RequestsLib.method("get", "post", "put", "delete",
                              "patch", "head", "request").where("verify", False)


@python_rule(
    id="PYTHON-LANG-SEC-054",
    name="Insecure HTTP Connection",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-319",
    tags="python,http,plaintext,insecure-transport,cwe-319",
    message="HTTPConnection used instead of HTTPSConnection. Use HTTPS for sensitive communications.",
    owasp="A02:2021",
)
def detect_http_connection():
    """Detects http.client.HTTPConnection usage."""
    return HttplibModule.method("HTTPConnection")
