"""
PYTHON-FLASK-SEC-008: Flask XSS via Raw HTML Concatenation
PYTHON-FLASK-SEC-015: Flask Unsanitized Input in Response

Security Impact: HIGH (SEC-008), MEDIUM (SEC-015)
CWE: CWE-79 (Improper Neutralization of Input During Web Page Generation)
OWASP: A03:2021 - Injection

DESCRIPTION:
These rules detect Cross-Site Scripting (XSS) vulnerabilities in Flask applications where
user-controlled input is reflected in HTTP responses without proper output encoding. Two
patterns are detected:

- **SEC-008**: Detects user input that flows into manually constructed HTML responses via
  make_response(). When developers build HTML strings by concatenating or formatting user
  data and return them as responses, they bypass Flask/Jinja2's automatic escaping and
  create reflected XSS vulnerabilities.

- **SEC-015**: Detects user input that flows directly into Flask response objects without
  sanitization. This is a broader pattern that catches cases where request data reaches
  make_response() or jsonify() without escaping, which may lead to XSS when the response
  is rendered in a browser context.

XSS allows attackers to inject malicious client-side scripts into web pages viewed by other
users. In Flask applications, Jinja2 templates provide automatic escaping by default, but
this protection is bypassed when developers construct HTML responses manually.

SECURITY IMPLICATIONS:

**1. Session Hijacking**:
Injected JavaScript can steal session cookies (document.cookie) and send them to an
attacker-controlled server, allowing complete account takeover.

**2. Credential Theft**:
Attackers can inject fake login forms or keyloggers that capture user credentials and
transmit them externally.

**3. Phishing and Content Spoofing**:
Page content can be arbitrarily modified to display misleading information, fake error
messages, or social engineering prompts that trick users into revealing sensitive data.

**4. Malware Distribution**:
Injected scripts can redirect users to malicious downloads or exploit browser
vulnerabilities to install malware.

VULNERABLE EXAMPLE:
```python
from flask import Flask, request, make_response

app = Flask(__name__)

@app.route('/greet')
def greet():
    name = request.args.get('name')
    html = "<h1>Hello " + name + "</h1>"
    return html

@app.route('/profile')
def profile():
    bio = request.form.get('bio')
    resp = make_response("<div>" + bio + "</div>")
    return resp
```

SECURE EXAMPLE:
```python
from flask import Flask, request, render_template
from markupsafe import escape

app = Flask(__name__)

@app.route('/greet')
def greet():
    name = request.args.get('name')
    # SAFE Option 1: Use Jinja2 template (auto-escapes by default)
    return render_template('greet.html', name=name)

@app.route('/greet-inline')
def greet_inline():
    name = request.args.get('name')
    # SAFE Option 2: Explicitly escape user input
    safe_name = escape(name)
    return f'<h1>Hello, {safe_name}!</h1>'
```

DETECTION AND PREVENTION:

**Pre-deployment checks**:
```bash
pathfinder scan --project . --ruleset cpf/python/PYTHON-FLASK-SEC-008,PYTHON-FLASK-SEC-015
```

**Code Review Checklist**:
- [ ] All user input in HTML responses is escaped via markupsafe.escape() or html.escape()
- [ ] Jinja2 templates with auto-escaping used instead of manual HTML construction
- [ ] No raw string concatenation of user input into HTML
- [ ] Content-Type headers set appropriately (application/json for API responses)
- [ ] Content-Security-Policy headers configured to restrict inline scripts

COMPLIANCE:
- CWE-79: Improper Neutralization of Input During Web Page Generation
- OWASP Top 10 A03:2021 - Injection
- SANS Top 25 (CWE-79 ranked #2)
- PCI DSS Requirement 6.5.7: Cross-Site Scripting

REFERENCES:
- CWE-79: https://cwe.mitre.org/data/definitions/79.html
- OWASP XSS: https://owasp.org/www-community/attacks/xss/
- OWASP XSS Prevention Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Scripting_Prevention_Cheat_Sheet.html
- Flask Security Considerations: https://flask.palletsprojects.com/en/latest/security/
- MarkupSafe: https://markupsafe.palletsprojects.com/

DETECTION SCOPE:
These rules perform inter-procedural taint analysis tracking data from Flask request sources
to make_response() and jsonify() sinks. Recognized sanitizers include escape(),
markupsafe.escape(), html.escape(), and bleach.clean().
"""

from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets


class FlaskModule(QueryType):
    fqns = ["flask"]


@python_rule(
    id="PYTHON-FLASK-SEC-008",
    name="Flask XSS via Raw HTML Concatenation",
    severity="HIGH",
    category="flask",
    cwe="CWE-79",
    tags="python,flask,xss,html,owasp-a07,cwe-79",
    message="User input concatenated into HTML response. Use render_template() or markupsafe.escape().",
    owasp="A07:2021",
)
def detect_flask_xss_html_concat():
    """Detects Flask request data concatenated into HTML and returned in response."""
    return flows(
        from_sources=[
            calls("request.args.get"),
            calls("request.form.get"),
            calls("request.values.get"),
            calls("request.get_json"),
        ],
        to_sinks=[
            FlaskModule.method("make_response").tracks(0),
            calls("make_response"),
        ],
        sanitized_by=[
            calls("escape"),
            calls("markupsafe.escape"),
            calls("html.escape"),
            calls("bleach.clean"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )


@python_rule(
    id="PYTHON-FLASK-SEC-015",
    name="Flask Unsanitized Input in Response",
    severity="MEDIUM",
    category="flask",
    cwe="CWE-79",
    tags="python,flask,xss,unsanitized-input,owasp-a07,cwe-79",
    message="User input returned directly in response without escaping. Use markupsafe.escape().",
    owasp="A07:2021",
)
def detect_flask_unsanitized_response():
    """Detects Flask request data returned directly in response."""
    return flows(
        from_sources=[
            calls("request.args.get"),
            calls("request.form.get"),
            calls("request.values.get"),
        ],
        to_sinks=[
            FlaskModule.method("make_response", "jsonify").tracks(0),
            calls("make_response"),
            calls("jsonify"),
        ],
        sanitized_by=[
            calls("escape"),
            calls("markupsafe.escape"),
            calls("html.escape"),
            calls("bleach.clean"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
