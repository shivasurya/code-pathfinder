"""
Python Django SSRF Rules

Rules:
- PYTHON-DJANGO-SEC-030: SSRF via requests Library (CWE-918)
- PYTHON-DJANGO-SEC-031: SSRF via urllib (CWE-918)

Security Impact: HIGH
CWE: CWE-918 (Server-Side Request Forgery)
OWASP: A10:2021 - Server-Side Request Forgery (SSRF)

DESCRIPTION:
These rules detect Server-Side Request Forgery (SSRF) vulnerabilities in Django applications
where untrusted user input from HTTP requests flows into server-side HTTP client calls such
as requests.get(), requests.post(), or urllib.request.urlopen(). SSRF allows attackers to
make the server send HTTP requests to arbitrary destinations, including internal services,
cloud metadata endpoints, and private network resources that are not directly accessible
from the internet.

SECURITY IMPLICATIONS:

**1. Internal Network Scanning**:
Attackers can probe internal network infrastructure by making the server send requests to
internal IP ranges (10.x.x.x, 172.16.x.x, 192.168.x.x), discovering internal services,
open ports, and application endpoints not exposed to the internet.

**2. Cloud Metadata Theft**:
On cloud platforms (AWS, GCP, Azure), attackers can access instance metadata endpoints
(e.g., http://169.254.169.254/latest/meta-data/) to steal IAM credentials, API keys,
and configuration data, potentially compromising the entire cloud account.

**3. Internal Service Access**:
SSRF enables attackers to interact with internal services (databases, caches, admin panels)
that lack authentication because they trust requests from the internal network.

**4. Data Exfiltration**:
Attackers can read internal files via file:// protocol or access internal APIs to extract
sensitive data, then relay it through the SSRF to external servers they control.

VULNERABLE EXAMPLE:
```python
import requests
import urllib.request


# SEC-030: SSRF via requests
def vulnerable_ssrf_requests(request):
    url = request.GET.get('url')
    resp = requests.get(url)
    return resp.text


# SEC-031: SSRF via urllib
def vulnerable_ssrf_urllib(request):
    url = request.POST.get('target')
    resp = urllib.request.urlopen(url)
    return resp.read()
```

SECURE EXAMPLE:
```python
from django.http import JsonResponse
import requests
from urllib.parse import urlparse
import ipaddress

ALLOWED_DOMAINS = {'api.example.com', 'cdn.example.com'}
BLOCKED_NETWORKS = [
    ipaddress.ip_network('10.0.0.0/8'),
    ipaddress.ip_network('172.16.0.0/12'),
    ipaddress.ip_network('192.168.0.0/16'),
    ipaddress.ip_network('169.254.0.0/16'),  # Cloud metadata
    ipaddress.ip_network('127.0.0.0/8'),     # Loopback
]

def is_safe_url(url):
    \"\"\"Validate URL against allowlist and block internal networks.\"\"\"
    parsed = urlparse(url)
    if parsed.scheme not in ('http', 'https'):
        return False
    if parsed.hostname in ALLOWED_DOMAINS:
        return True
    try:
        ip = ipaddress.ip_address(parsed.hostname)
        return not any(ip in network for network in BLOCKED_NETWORKS)
    except ValueError:
        return False

def fetch_url(request):
    # SECURE: Validate URL before making request
    url = request.GET.get('url', '')
    if not is_safe_url(url):
        return JsonResponse({'error': 'URL not allowed'}, status=400)
    response = requests.get(url, timeout=5)
    return JsonResponse({'content': response.text})
```

DETECTION AND PREVENTION:

**Key Mitigation Strategies**:
- Maintain a strict allowlist of permitted domains and URL schemes
- Block requests to internal/private IP ranges and cloud metadata endpoints
- Resolve DNS and validate the resolved IP address (not just the hostname) to prevent DNS rebinding
- Use network-level controls (firewall rules) to restrict outbound traffic from the application
- Disable unnecessary URL schemes (file://, gopher://, dict://)
- Set timeouts on all outbound HTTP requests to prevent resource exhaustion

**Pre-deployment checks**:
```bash
pathfinder scan --project . --ruleset cpf/python/django/ssrf
```

COMPLIANCE:
- CWE-918: Server-Side Request Forgery (SSRF)
- OWASP A10:2021 - Server-Side Request Forgery (SSRF)
- SANS Top 25: CWE-918
- NIST SP 800-53: SC-7 (Boundary Protection)

REFERENCES:
- CWE-918: https://cwe.mitre.org/data/definitions/918.html
- OWASP SSRF: https://owasp.org/www-community/attacks/Server_Side_Request_Forgery
- OWASP SSRF Prevention Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html
- Django Security: https://docs.djangoproject.com/en/stable/topics/security/
"""

from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets


class RequestsLib(QueryType):
    fqns = ["requests"]


class UrllibModule(QueryType):
    fqns = ["urllib.request"]


_DJANGO_SOURCES = [
    calls("request.GET.get"),
    calls("request.POST.get"),
    calls("request.GET"),
    calls("request.POST"),
    calls("request.COOKIES.get"),
    calls("request.FILES.get"),
    calls("*.GET.get"),
    calls("*.POST.get"),
]


@python_rule(
    id="PYTHON-DJANGO-SEC-030",
    name="Django SSRF via requests Library",
    severity="HIGH",
    category="django",
    cwe="CWE-918",
    tags="python,django,ssrf,requests,owasp-a10,cwe-918",
    message="User input flows to requests HTTP call. Validate and restrict URLs.",
    owasp="A10:2021",
)
def detect_django_ssrf_requests():
    """Detects Django request data flowing to requests library calls."""
    return flows(
        from_sources=_DJANGO_SOURCES,
        to_sinks=[
            RequestsLib.method("get", "post", "put", "delete", "patch",
                               "head", "options", "request").tracks(0),
        ],
        sanitized_by=[
            calls("urllib.parse.urlparse"),
            calls("validators.url"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )


@python_rule(
    id="PYTHON-DJANGO-SEC-031",
    name="Django SSRF via urllib",
    severity="HIGH",
    category="django",
    cwe="CWE-918",
    tags="python,django,ssrf,urllib,owasp-a10,cwe-918",
    message="User input flows to urllib.request.urlopen(). Validate and restrict URLs.",
    owasp="A10:2021",
)
def detect_django_ssrf_urllib():
    """Detects Django request data flowing to urllib calls."""
    return flows(
        from_sources=_DJANGO_SOURCES,
        to_sinks=[
            UrllibModule.method("urlopen", "Request").tracks(0),
        ],
        sanitized_by=[
            calls("urllib.parse.urlparse"),
            calls("validators.url"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
