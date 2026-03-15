"""
PYTHON-FLASK-SEC-006: Flask SSRF via requests library

Security Impact: HIGH
CWE: CWE-918 (Server-Side Request Forgery)
OWASP: A10:2021 - Server-Side Request Forgery

DESCRIPTION:
This rule detects Server-Side Request Forgery (SSRF) vulnerabilities in Flask applications
where user-controlled input from HTTP request parameters flows into outbound HTTP request
URLs. When an attacker controls the destination URL of a server-initiated HTTP request,
they can force the application server to make requests to arbitrary internal or external
endpoints.

SSRF is particularly dangerous in cloud environments where metadata endpoints
(e.g., http://169.254.169.254/) expose instance credentials, and in microservice
architectures where internal services trust requests originating from the local network.

SECURITY IMPLICATIONS:

**1. Internal Service Access**:
Attackers can probe and interact with internal services that are not directly accessible
from the internet, including databases, caches, admin panels, and monitoring systems
bound to localhost or internal network addresses.

**2. Cloud Metadata Exploitation**:
In AWS, GCP, and Azure environments, SSRF can be used to query instance metadata services
to steal IAM credentials, service account tokens, and infrastructure configuration details.

**3. Port Scanning and Service Discovery**:
Attackers can use the vulnerable server as a proxy to scan internal networks, discover
services, and map infrastructure topology by observing response timing and error messages.

**4. Data Exfiltration via Protocol Smuggling**:
Depending on the HTTP library used, attackers may be able to use non-HTTP protocols
(file://, gopher://, dict://) to read local files or interact with services like Redis
and Memcached.

VULNERABLE EXAMPLE:
```python
from flask import Flask, request
import requests

app = Flask(__name__)

@app.route('/fetch')
def fetch_url():
    url = request.args.get('url')
    # VULNERABLE: User controls the outbound request URL
    response = requests.get(url)
    return response.text

# Attack: GET /fetch?url=http://169.254.169.254/latest/meta-data/iam/security-credentials/
# Attack: GET /fetch?url=http://localhost:6379/INFO
```

SECURE EXAMPLE:
```python
from flask import Flask, request
from urllib.parse import urlparse
import requests
import ipaddress

ALLOWED_HOSTS = {'api.example.com', 'cdn.example.com'}

def validate_url(url):
    parsed = urlparse(url)
    if parsed.scheme not in ('http', 'https'):
        raise ValueError('Invalid scheme')
    if parsed.hostname not in ALLOWED_HOSTS:
        raise ValueError('Host not in allowlist')
    # Block private/reserved IP ranges
    try:
        ip = ipaddress.ip_address(parsed.hostname)
        if ip.is_private or ip.is_loopback or ip.is_link_local:
            raise ValueError('Private IP not allowed')
    except ValueError:
        pass  # hostname is not an IP, DNS will resolve it
    return url

app = Flask(__name__)

@app.route('/fetch')
def fetch_url():
    url = request.args.get('url')
    # SAFE: URL validated against allowlist before use
    safe_url = validate_url(url)
    response = requests.get(safe_url)
    return response.text
```

DETECTION AND PREVENTION:

**Pre-deployment checks**:
```bash
pathfinder scan --project . --ruleset cpf/python/PYTHON-FLASK-SEC-006
```

**Code Review Checklist**:
- [ ] No user input flows directly to HTTP request URLs
- [ ] URL scheme restricted to http/https only
- [ ] Hostname validated against an explicit allowlist
- [ ] Private and reserved IP ranges blocked (127.0.0.0/8, 10.0.0.0/8, 169.254.0.0/16)
- [ ] DNS rebinding protections in place (resolve hostname before validation)
- [ ] Redirect following disabled or limited for outbound requests

COMPLIANCE:
- CWE-918: Server-Side Request Forgery (SSRF)
- OWASP Top 10 A10:2021 - Server-Side Request Forgery
- NIST SP 800-53 SC-7: Boundary Protection

REFERENCES:
- CWE-918: https://cwe.mitre.org/data/definitions/918.html
- OWASP SSRF: https://owasp.org/www-community/attacks/Server_Side_Request_Forgery
- OWASP SSRF Prevention Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html
- Cloud SSRF Exploitation: https://owasp.org/www-project-web-security-testing-guide/latest/4-Web_Application_Security_Testing/07-Input_Validation_Testing/19-Testing_for_Server-Side_Request_Forgery

DETECTION SCOPE:
This rule performs inter-procedural taint analysis tracking data from Flask request sources
to requests library methods (get, post, put, delete, etc.) and urllib.request.urlopen sinks.
Recognized sanitizers include validate_url() and is_safe_url() methods.
"""

from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets


class RequestsLib(QueryType):
    fqns = ["requests"]


class UrllibRequest(QueryType):
    fqns = ["urllib.request"]


@python_rule(
    id="PYTHON-FLASK-SEC-006",
    name="Flask SSRF via requests library",
    severity="HIGH",
    category="flask",
    cwe="CWE-918",
    tags="python,flask,ssrf,requests,owasp-a10,cwe-918",
    message="User input flows to HTTP request URL. Validate and allowlist target URLs.",
    owasp="A10:2021",
)
def detect_flask_ssrf():
    """Detects Flask request data flowing to requests library calls."""
    return flows(
        from_sources=[
            calls("request.args.get"),
            calls("request.form.get"),
            calls("request.values.get"),
            calls("request.get_json"),
        ],
        to_sinks=[
            RequestsLib.method("get", "post", "put", "delete", "patch",
                               "head", "options", "request").tracks(0),
            UrllibRequest.method("urlopen", "Request").tracks(0),
            calls("http_requests.get"),
            calls("http_requests.post"),
        ],
        sanitized_by=[
            calls("*.validate_url"),
            calls("*.is_safe_url"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
