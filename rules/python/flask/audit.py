"""
Flask Audit Rules - Configuration and Pattern-Based Security Detection

PYTHON-FLASK-AUDIT-003: Flask Bound to All Interfaces (CWE-200)
PYTHON-FLASK-AUDIT-004: Flask CORS Wildcard Origin (CWE-942)
PYTHON-FLASK-AUDIT-005: Flask url_for with _external=True (CWE-601)
PYTHON-FLASK-AUDIT-008: Flask render_template_string Usage (CWE-96)
PYTHON-FLASK-AUDIT-009: Flask Cookie Without Secure Flags (CWE-614)
PYTHON-FLASK-AUDIT-010: Flask WTF CSRF Disabled (CWE-352)
PYTHON-FLASK-SEC-017: Flask Insecure Static File Serve (CWE-22)
PYTHON-FLASK-SEC-018: Flask Hashids with Secret Key (CWE-330)
PYTHON-FLASK-XSS-001: Flask Direct Use of Jinja2 (CWE-79)
PYTHON-FLASK-XSS-002: Flask Explicit Unescape with Markup (CWE-79)

Security Impact: LOW to HIGH (varies by rule)
OWASP: A01:2021, A02:2021, A03:2021, A05:2021, A07:2021

DESCRIPTION:
This module contains ten audit and configuration-based security rules for Flask applications.
Unlike dataflow rules that track tainted input from sources to sinks, these rules use pattern
matching to detect insecure configurations, dangerous API usage, and security anti-patterns
that do not require taint analysis.

These rules serve two purposes: (1) detecting definitive security misconfigurations that should
be fixed regardless of context, and (2) flagging potentially dangerous patterns that warrant
manual review to determine if they represent actual vulnerabilities in the specific application
context.

RULES OVERVIEW:

**AUDIT-003 - Bound to All Interfaces (CWE-200, MEDIUM)**:
Detects app.run(host='0.0.0.0') which binds the Flask development server to all network
interfaces, potentially exposing it to the public internet. The Flask development server
is not designed for production use and lacks security hardening. Binding to 0.0.0.0 in
development environments can expose debug endpoints, and in production indicates the
development server is being used inappropriately.

**AUDIT-004 - CORS Wildcard Origin (CWE-942, MEDIUM)**:
Detects Flask-CORS configured with origins='*' which allows any website to make
cross-origin requests to the application. This can expose sensitive API endpoints to
malicious third-party websites, enabling cross-origin data theft if credentials are
included. When combined with supports_credentials=True, this dynamically reflects the
Origin header, effectively disabling the Same-Origin Policy.

**AUDIT-005 - url_for External URLs (CWE-601, LOW)**:
Detects url_for(_external=True) which generates absolute URLs using the Host header from
the incoming HTTP request. If an attacker controls the Host header (via header injection
or misconfigured reverse proxies), the generated URLs point to attacker-controlled domains,
enabling phishing attacks or OAuth token theft.

**AUDIT-008 - render_template_string Usage (CWE-96, MEDIUM)**:
Flags any usage of render_template_string() as an audit finding. While not inherently
vulnerable when used with only hardcoded template strings, this function is the primary
vector for Server-Side Template Injection (SSTI) in Flask and its presence warrants review
to ensure no user input reaches the template string parameter.

**AUDIT-009 - Insecure Cookie Flags (CWE-614, MEDIUM)**:
Detects set_cookie() calls with secure=False or httponly=False. Without the Secure flag,
cookies are transmitted over unencrypted HTTP connections where they can be intercepted.
Without the HttpOnly flag, cookies are accessible to JavaScript, enabling theft via XSS.
Session cookies and authentication tokens must always set both flags.

**AUDIT-010 - CSRF Protection Disabled (CWE-352, HIGH)**:
Detects WTF_CSRF_ENABLED being set to False in Flask-WTF configuration. Disabling CSRF
protection allows attackers to forge cross-site requests that perform state-changing
operations (transfers, password changes, data deletion) on behalf of authenticated users
who visit attacker-controlled pages.

**SEC-017 - Insecure Static File Serve (CWE-22, MEDIUM)**:
Flags usage of send_from_directory() and send_file() which serve files from the file system.
When filenames are derived from user input without sanitization via secure_filename(), path
traversal sequences can escape the intended directory and expose arbitrary server files.

**SEC-018 - Hashids with Secret Key (CWE-330, MEDIUM)**:
Detects Hashids(salt=app.secret_key) where the Flask SECRET_KEY is reused as the Hashids
salt. The Hashids algorithm is not cryptographically secure -- the salt can be recovered by
analyzing a sufficient number of generated hashes. This recovery compromises the Flask
SECRET_KEY, enabling session forgery, cookie tampering, and CSRF token prediction.

**XSS-001 - Direct Jinja2 Environment (CWE-79, MEDIUM)**:
Detects direct instantiation of jinja2.Environment which creates a template environment
outside of Flask's managed context. Flask automatically configures Jinja2 with
autoescape=True for HTML templates, but directly created Environment instances use
autoescape=False by default, potentially introducing XSS vulnerabilities.

**XSS-002 - Markup Bypass (CWE-79, MEDIUM)**:
Detects usage of Markup() or markupsafe.Markup() which marks a string as safe HTML,
bypassing Jinja2's automatic escaping. If user-controlled data is wrapped in Markup()
without prior sanitization, it creates an XSS vulnerability by explicitly telling the
template engine not to escape the content.

SECURITY IMPLICATIONS:

**Configuration Weaknesses** (AUDIT-003, AUDIT-004, AUDIT-009, AUDIT-010):
Misconfigured security settings weaken the application's defense-in-depth posture,
exposing it to network-level attacks, cross-origin exploitation, session hijacking, and
cross-site request forgery.

**Dangerous API Patterns** (AUDIT-005, AUDIT-008, SEC-017, XSS-001, XSS-002):
Usage of security-sensitive APIs warrants review to ensure proper safeguards are in place.
These patterns are not always vulnerable but represent common sources of security issues.

**Cryptographic Weakness** (SEC-018):
Reusing the Flask secret key as a Hashids salt creates a side channel through which the
application's master secret can be recovered, compromising all cryptographic operations
that depend on it.

VULNERABLE EXAMPLE:
```python
from flask import Flask, make_response
from flask_cors import CORS
from jinja2 import Environment
from markupsafe import Markup

app = Flask(__name__)

# Debug mode enabled
app.run(debug=True)

# Bind to all interfaces
app.run(host="0.0.0.0")

# CORS wildcard
CORS(app, origins="*")

# Cookie without secure flag
@app.route('/setcookie')
def setcookie():
    resp = make_response("cookie set")
    resp.set_cookie('session', 'value', secure=False, httponly=False)
    return resp

# Direct Jinja2 usage without autoescape
env = Environment(autoescape=False)

# Markup usage
html = Markup("<b>hello</b>")
```

DETECTION AND PREVENTION:

**Pre-deployment checks**:
```bash
pathfinder scan --project . --ruleset cpf/python/flask/audit
```

**Code Review Checklist**:
- [ ] Flask app not bound to 0.0.0.0 in production
- [ ] CORS origins restricted to specific trusted domains
- [ ] url_for(_external=True) only used with trusted Host header configuration
- [ ] render_template_string() not used with any user-supplied template content
- [ ] All cookies set with secure=True, httponly=True, samesite='Lax'
- [ ] CSRF protection enabled (WTF_CSRF_ENABLED not set to False)
- [ ] send_from_directory/send_file use secure_filename() for user-supplied names
- [ ] Flask SECRET_KEY not reused as Hashids salt
- [ ] Jinja2 Environment created with autoescape=True for HTML output
- [ ] Markup() only wraps trusted, pre-sanitized content

COMPLIANCE:
- CWE-200: Exposure of Sensitive Information to an Unauthorized Actor
- CWE-942: Permissive Cross-domain Policy with Untrusted Domains
- CWE-601: URL Redirection to Untrusted Site
- CWE-96: Improper Neutralization of Directives in Statically Saved Code
- CWE-614: Sensitive Cookie in HTTPS Session Without 'Secure' Attribute
- CWE-352: Cross-Site Request Forgery (CSRF)
- CWE-22: Improper Limitation of a Pathname to a Restricted Directory
- CWE-330: Use of Insufficiently Random Values
- CWE-79: Improper Neutralization of Input During Web Page Generation
- OWASP Top 10 A01:2021 - Broken Access Control
- OWASP Top 10 A02:2021 - Cryptographic Failures
- OWASP Top 10 A03:2021 - Injection
- OWASP Top 10 A05:2021 - Security Misconfiguration
- OWASP Top 10 A07:2021 - Cross-Site Scripting

REFERENCES:
- CWE-200: https://cwe.mitre.org/data/definitions/200.html
- CWE-942: https://cwe.mitre.org/data/definitions/942.html
- CWE-614: https://cwe.mitre.org/data/definitions/614.html
- CWE-352: https://cwe.mitre.org/data/definitions/352.html
- CWE-330: https://cwe.mitre.org/data/definitions/330.html
- CWE-79: https://cwe.mitre.org/data/definitions/79.html
- Flask Security Considerations: https://flask.palletsprojects.com/en/latest/security/
- Flask-CORS Documentation: https://flask-cors.readthedocs.io/
- Flask-WTF CSRF: https://flask-wtf.readthedocs.io/en/latest/csrf/
- Hashids Cryptanalysis: http://carnage.github.io/2015/08/cryptanalysis-of-hashids
- OWASP Secure Cookie Attributes: https://owasp.org/www-community/controls/SecureCookieAttribute
- Jinja2 Autoescaping: https://jinja.palletsprojects.com/en/latest/api/#autoescaping

DETECTION SCOPE:
These rules use pattern matching (call site analysis) rather than taint tracking. They detect
specific API calls and configuration patterns without requiring dataflow analysis, making them
fast and suitable for broad codebase auditing.
"""

from rules.python_decorators import python_rule
from codepathfinder import calls, Or, QueryType


class FlaskApp(QueryType):
    fqns = ["flask"]


class FlaskCORS(QueryType):
    fqns = ["flask_cors"]


class HashidsModule(QueryType):
    fqns = ["hashids"]


@python_rule(
    id="PYTHON-FLASK-AUDIT-003",
    name="Flask Bound to All Interfaces",
    severity="MEDIUM",
    category="flask",
    cwe="CWE-200",
    tags="python,flask,network,binding,cwe-200",
    message="Flask app bound to 0.0.0.0 (all interfaces). Bind to 127.0.0.1 in production.",
    owasp="A05:2021",
)
def detect_flask_bind_all():
    """Detects app.run(host='0.0.0.0')."""
    return FlaskApp.method("run").where("host", "0.0.0.0")


@python_rule(
    id="PYTHON-FLASK-AUDIT-004",
    name="Flask CORS Wildcard Origin",
    severity="MEDIUM",
    category="flask",
    cwe="CWE-942",
    tags="python,flask,cors,wildcard,cwe-942",
    message="CORS configured with wildcard origin '*'. Restrict to specific domains.",
    owasp="A05:2021",
)
def detect_flask_cors_wildcard():
    """Detects CORS(app, origins='*')."""
    return FlaskCORS.method("CORS").where("origins", "*")


@python_rule(
    id="PYTHON-FLASK-AUDIT-005",
    name="Flask url_for with _external=True",
    severity="LOW",
    category="flask",
    cwe="CWE-601",
    tags="python,flask,url-for,external,cwe-601",
    message="url_for() with _external=True may expose internal URL schemes. Verify usage.",
    owasp="A01:2021",
)
def detect_flask_url_for_external():
    """Detects url_for(_external=True)."""
    return calls("url_for", match_name={"_external": True})


@python_rule(
    id="PYTHON-FLASK-AUDIT-008",
    name="Flask render_template_string Usage",
    severity="MEDIUM",
    category="flask",
    cwe="CWE-96",
    tags="python,flask,template,ssti,audit,cwe-96",
    message="render_template_string() detected. Prefer render_template() with separate template files.",
    owasp="A03:2021",
)
def detect_flask_render_template_string():
    """Audit: Detects any usage of render_template_string()."""
    return Or(
        calls("render_template_string"),
        calls("flask.render_template_string"),
    )


@python_rule(
    id="PYTHON-FLASK-AUDIT-009",
    name="Flask Cookie Without Secure Flags",
    severity="MEDIUM",
    category="flask",
    cwe="CWE-614",
    tags="python,flask,cookie,secure,httponly,cwe-614",
    message="Cookie set without secure=True or httponly=True. Set both flags for session cookies.",
    owasp="A05:2021",
)
def detect_flask_insecure_cookie():
    """Detects set_cookie() with secure=False or httponly=False."""
    return Or(
        calls("*.set_cookie", match_name={"secure": False}),
        calls("*.set_cookie", match_name={"httponly": False}),
    )


@python_rule(
    id="PYTHON-FLASK-AUDIT-010",
    name="Flask WTF CSRF Disabled",
    severity="HIGH",
    category="flask",
    cwe="CWE-352",
    tags="python,flask,csrf,wtf,cwe-352",
    message="WTF_CSRF_ENABLED set to False. CSRF protection should always be enabled.",
    owasp="A05:2021",
)
def detect_flask_wtf_csrf_disabled():
    """Detects WTF_CSRF_ENABLED = False. Pattern match on config assignment."""
    return calls("*.config.__setitem__", match_position={0: "WTF_CSRF_ENABLED"})


@python_rule(
    id="PYTHON-FLASK-SEC-017",
    name="Flask Insecure Static File Serve",
    severity="MEDIUM",
    category="flask",
    cwe="CWE-22",
    tags="python,flask,path-traversal,static-files,cwe-22",
    message="send_from_directory() with user input. Use werkzeug.utils.secure_filename().",
    owasp="A01:2021",
)
def detect_flask_insecure_static_serve():
    """Detects send_from_directory() usage (audit for user-controlled filename)."""
    return Or(
        calls("send_from_directory"),
        calls("flask.send_from_directory"),
        calls("send_file"),
        calls("flask.send_file"),
    )


@python_rule(
    id="PYTHON-FLASK-SEC-018",
    name="Flask Hashids with Secret Key",
    severity="MEDIUM",
    category="flask",
    cwe="CWE-330",
    tags="python,flask,hashids,secret-key,cwe-330",
    message="Flask SECRET_KEY used as Hashids salt. Use a separate salt value.",
    owasp="A02:2021",
)
def detect_flask_hashids_secret():
    """Detects Hashids(salt=app.secret_key)."""
    return HashidsModule.method("Hashids").where("salt", "app.secret_key")


@python_rule(
    id="PYTHON-FLASK-XSS-001",
    name="Flask Direct Use of Jinja2",
    severity="MEDIUM",
    category="flask",
    cwe="CWE-79",
    tags="python,flask,jinja2,xss,audit,cwe-79",
    message="Direct Jinja2 Environment usage bypasses Flask's auto-escaping. Use Flask's render_template().",
    owasp="A07:2021",
)
def detect_flask_direct_jinja2():
    """Detects direct Jinja2 Environment creation."""
    return Or(
        calls("Environment"),
        calls("jinja2.Environment"),
    )


@python_rule(
    id="PYTHON-FLASK-XSS-002",
    name="Flask Explicit Unescape with Markup",
    severity="MEDIUM",
    category="flask",
    cwe="CWE-79",
    tags="python,flask,markup,xss,audit,cwe-79",
    message="Markup() bypasses auto-escaping. Ensure input is trusted before wrapping in Markup().",
    owasp="A07:2021",
)
def detect_flask_markup_usage():
    """Detects Markup() usage which bypasses escaping."""
    return Or(
        calls("Markup"),
        calls("markupsafe.Markup"),
    )
