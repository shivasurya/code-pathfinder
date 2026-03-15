"""
Insecure Transport and Cleartext Transmission Security Rules for Python

Rules in this file:
- PYTHON-LANG-SEC-060: HTTP Request Without TLS (CWE-319)
- PYTHON-LANG-SEC-061: Insecure urllib.urlopen (CWE-319)
- PYTHON-LANG-SEC-062: Insecure urllib Request Object (CWE-319)
- PYTHON-LANG-SEC-063: FTP Without TLS (CWE-319)
- PYTHON-LANG-SEC-064: telnetlib Usage Detected (CWE-319)

Security Impact: MEDIUM
CWE: CWE-319 (Cleartext Transmission of Sensitive Information)
OWASP: A02:2021 - Cryptographic Failures

DESCRIPTION:
Transmitting sensitive data over unencrypted channels (HTTP, FTP, Telnet) exposes
it to interception by network-level attackers through traffic sniffing,
man-in-the-middle (MITM) attacks, or ARP spoofing. Python's requests library,
urllib module, ftplib, and telnetlib can all be configured to communicate over
plaintext protocols that provide no confidentiality or integrity guarantees.

SECURITY IMPLICATIONS:
When HTTP is used instead of HTTPS, all request and response data -- including
authentication credentials, session tokens, API keys, and personal data --
travels in plaintext across every network hop. An attacker on the same network
segment (Wi-Fi, corporate LAN, ISP) can passively capture this data with basic
packet sniffing tools. FTP transmits credentials and file contents in cleartext.
Telnet provides no encryption whatsoever, transmitting every keystroke including
passwords as plaintext.

    # Attack scenario: credential theft via HTTP sniffing
    # Attacker on same WiFi captures this request with Wireshark
    requests.post("http://api.example.com/login",
                  data={"user": "admin", "pass": "secret123"})

VULNERABLE EXAMPLE:
```python
import requests, ftplib, telnetlib
# HTTP instead of HTTPS
response = requests.get("http://api.example.com/users")
# FTP without TLS encryption
ftp = ftplib.FTP("ftp.example.com")
ftp.login("user", "password")  # Credentials in cleartext
# Telnet with no encryption
tn = telnetlib.Telnet("server.example.com")
```

SECURE EXAMPLE:
```python
import requests, ftplib
# Always use HTTPS for network requests
response = requests.get("https://api.example.com/users")
# Use FTP over TLS (FTPS)
ftp = ftplib.FTP_TLS("ftp.example.com")
ftp.auth()  # Establish TLS
ftp.login("user", "password")  # Encrypted
# Use SSH (paramiko) instead of telnet
import paramiko
ssh = paramiko.SSHClient()
ssh.connect("server.example.com", username="user", key_filename="key.pem")
```

DETECTION AND PREVENTION:
- Enforce HTTPS for all HTTP requests by validating URL schemes
- Replace ftplib.FTP with ftplib.FTP_TLS for encrypted file transfers
- Replace telnetlib with SSH-based connections (paramiko or asyncssh)
- Implement network-level controls (HSTS headers, TLS-only load balancers)
- Use URL validation middleware to reject http:// schemes in production

COMPLIANCE:
- CWE-319: Cleartext Transmission of Sensitive Information
- OWASP A02:2021 - Cryptographic Failures
- PCI DSS v4.0 Requirement 4.2: Strong cryptography for data transmission
- NIST SP 800-52 Rev 2: Guidelines for TLS Implementations
- SANS Top 25 (2023) - CWE-319: Cleartext Transmission

REFERENCES:
- https://cwe.mitre.org/data/definitions/319.html
- https://owasp.org/Top10/A02_2021-Cryptographic_Failures/
- https://docs.python.org/3/library/ftplib.html#ftplib.FTP_TLS
- https://docs.python.org/3/library/http.client.html#http.client.HTTPSConnection
- https://requests.readthedocs.io/en/latest/user/advanced/#ssl-cert-verification
"""

from rules.python_decorators import python_rule
from codepathfinder import calls, QueryType


class RequestsLib(QueryType):
    fqns = ["requests"]


class UrllibModule(QueryType):
    fqns = ["urllib.request"]


class FtplibModule(QueryType):
    fqns = ["ftplib"]


class TelnetModule(QueryType):
    fqns = ["telnetlib"]


@python_rule(
    id="PYTHON-LANG-SEC-060",
    name="HTTP Request Without TLS",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-319",
    tags="python,requests,http,insecure-transport,cwe-319",
    message="HTTP URL used in requests call. Use HTTPS for sensitive data transmission.",
    owasp="A02:2021",
)
def detect_requests_http():
    """Detects requests library calls (audit for HTTP URLs)."""
    return RequestsLib.method("get", "post", "put", "delete", "patch", "head", "request")


@python_rule(
    id="PYTHON-LANG-SEC-061",
    name="Insecure urllib.urlopen",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-319",
    tags="python,urllib,http,insecure-transport,cwe-319",
    message="urllib.request.urlopen() detected. Ensure HTTPS URLs are used.",
    owasp="A02:2021",
)
def detect_urllib_insecure():
    """Detects urllib.request.urlopen and urlretrieve calls."""
    return UrllibModule.method("urlopen", "urlretrieve")


@python_rule(
    id="PYTHON-LANG-SEC-062",
    name="Insecure urllib Request Object",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-319",
    tags="python,urllib,request-object,insecure-transport,cwe-319",
    message="urllib.request.Request() detected. Ensure HTTPS URLs are used.",
    owasp="A02:2021",
)
def detect_urllib_request():
    """Detects urllib.request.Request and OpenerDirector usage."""
    return UrllibModule.method("Request", "OpenerDirector")


@python_rule(
    id="PYTHON-LANG-SEC-063",
    name="FTP Without TLS",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-319",
    tags="python,ftp,insecure-transport,cwe-319",
    message="ftplib.FTP() without TLS. Use ftplib.FTP_TLS() instead.",
    owasp="A02:2021",
)
def detect_ftp_no_tls():
    """Detects ftplib.FTP usage without TLS."""
    return FtplibModule.method("FTP")


@python_rule(
    id="PYTHON-LANG-SEC-064",
    name="telnetlib Usage Detected",
    severity="HIGH",
    category="lang",
    cwe="CWE-319",
    tags="python,telnet,insecure-transport,plaintext,cwe-319",
    message="telnetlib transmits data in plaintext. Use SSH (paramiko) instead.",
    owasp="A02:2021",
)
def detect_telnet():
    """Detects telnetlib.Telnet usage."""
    return TelnetModule.method("Telnet")
