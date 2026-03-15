"""
PYTHON-FLASK-SEC-012: Flask Open Redirect

Security Impact: MEDIUM
CWE: CWE-601 (URL Redirection to Untrusted Site)
OWASP: A01:2021 - Broken Access Control

DESCRIPTION:
This rule detects open redirect vulnerabilities in Flask applications where user-controlled
input flows into redirect() function calls without URL validation. An open redirect occurs
when an application accepts a user-supplied URL or path and uses it as the destination of an
HTTP redirect response (3xx status code) without verifying that the target URL belongs to a
trusted domain.

Open redirects are commonly found in login flows (redirect after authentication), logout
handlers, and link shortener patterns. While sometimes considered low severity, they are
a critical component in phishing attack chains and can be combined with other vulnerabilities
for greater impact.

SECURITY IMPLICATIONS:

**1. Phishing Attacks**:
Attackers craft URLs pointing to the legitimate application domain with a redirect parameter
targeting a malicious phishing site. Victims see the trusted domain in the URL and are more
likely to enter credentials on the attacker's fake login page.

**2. OAuth Token Theft**:
In OAuth flows, open redirects can be exploited to steal authorization codes or access tokens
by redirecting the callback to an attacker-controlled endpoint.

**3. Credential Harvesting**:
Combined with look-alike domains, open redirects create convincing phishing URLs:
`https://trusted-app.com/login?next=https://trusted-app.evil.com/login`

**4. Bypassing Security Filters**:
Open redirects on trusted domains can bypass URL-based security filters, email link
scanners, and web application firewalls that allowlist the trusted domain.

VULNERABLE EXAMPLE:
```python
from flask import Flask, request, redirect

app = Flask(__name__)

@app.route('/login', methods=['POST'])
def login():
    # ... authenticate user ...
    next_url = request.args.get('next')
    # VULNERABLE: User controls redirect destination
    return redirect(next_url)

# Attack: POST /login?next=https://evil-phishing-site.com/fake-login
# User authenticates successfully, then gets redirected to attacker's site
```

SECURE EXAMPLE:
```python
from flask import Flask, request, redirect, url_for
from urllib.parse import urlparse

app = Flask(__name__)

def is_safe_redirect_url(target):
    \"\"\"Verify the redirect URL is relative or points to the same host.\"\"\"
    host_url = urlparse(request.host_url)
    redirect_url = urlparse(target)
    return (redirect_url.scheme in ('', 'http', 'https') and
            redirect_url.netloc in ('', host_url.netloc))

@app.route('/login', methods=['POST'])
def login():
    # ... authenticate user ...
    next_url = request.args.get('next')
    # SAFE: Validate redirect URL before use
    if next_url and is_safe_redirect_url(next_url):
        return redirect(next_url)
    return redirect(url_for('index'))  # Default to known safe route
```

DETECTION AND PREVENTION:

**Pre-deployment checks**:
```bash
pathfinder scan --project . --ruleset cpf/python/PYTHON-FLASK-SEC-012
```

**Code Review Checklist**:
- [ ] No user input flows directly to redirect() without validation
- [ ] Redirect URLs validated to be relative paths or same-host URLs
- [ ] url_for() used for internal redirects instead of raw URL strings
- [ ] url_has_allowed_host_and_scheme() used for Django-style validation
- [ ] Protocol-relative URLs (//) handled in validation logic

COMPLIANCE:
- CWE-601: URL Redirection to Untrusted Site (Open Redirect)
- OWASP Top 10 A01:2021 - Broken Access Control
- NIST SP 800-53 SI-10: Information Input Validation

REFERENCES:
- CWE-601: https://cwe.mitre.org/data/definitions/601.html
- OWASP Unvalidated Redirects: https://cheatsheetseries.owasp.org/cheatsheets/Unvalidated_Redirects_and_Forwards_Cheat_Sheet.html
- Flask redirect(): https://flask.palletsprojects.com/en/latest/api/#flask.redirect
- Flask url_for(): https://flask.palletsprojects.com/en/latest/api/#flask.url_for

DETECTION SCOPE:
This rule performs inter-procedural taint analysis tracking data from Flask request sources
to redirect() sinks. Recognized sanitizers include url_for() and
url_has_allowed_host_and_scheme().
"""

from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets


class FlaskModule(QueryType):
    fqns = ["flask"]


@python_rule(
    id="PYTHON-FLASK-SEC-012",
    name="Flask Open Redirect",
    severity="MEDIUM",
    category="flask",
    cwe="CWE-601",
    tags="python,flask,open-redirect,owasp-a01,cwe-601",
    message="User input flows to redirect(). Validate redirect URLs against an allowlist.",
    owasp="A01:2021",
)
def detect_flask_open_redirect():
    """Detects Flask request data flowing to redirect()."""
    return flows(
        from_sources=[
            calls("request.args.get"),
            calls("request.form.get"),
            calls("request.values.get"),
            calls("request.get_json"),
        ],
        to_sinks=[
            FlaskModule.method("redirect").tracks(0),
            calls("redirect"),
        ],
        sanitized_by=[
            calls("url_for"),
            calls("url_has_allowed_host_and_scheme"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
