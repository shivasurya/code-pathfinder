"""
PYTHON-FLASK-SEC-014: Flask Server-Side Template Injection (SSTI)

Security Impact: CRITICAL
CWE: CWE-96 (Improper Neutralization of Directives in Statically Saved Code)
OWASP: A03:2021 - Injection

DESCRIPTION:
This rule detects Server-Side Template Injection (SSTI) vulnerabilities in Flask applications
where user-controlled input flows into render_template_string(). Unlike render_template()
which loads templates from files, render_template_string() compiles and renders a Jinja2
template from a string argument. When user input is embedded in that template string,
attackers can inject Jinja2 template directives that execute arbitrary Python code on the
server.

SSTI in Jinja2 is especially dangerous because the template engine provides access to
Python's object hierarchy. Attackers can traverse from basic objects through __class__,
__mro__, __subclasses__(), and __globals__ to reach arbitrary Python functions including
os.system(), subprocess.Popen(), and import mechanisms.

SECURITY IMPLICATIONS:

**1. Remote Code Execution**:
Jinja2 template expressions can access Python's introspection capabilities to execute
arbitrary code. The classic payload `{{''.__class__.__mro__[1].__subclasses__()}}` enumerates
all loaded Python classes, from which attackers select one with access to os or subprocess.

**2. Information Disclosure**:
Template expressions like `{{config}}` directly expose the Flask application's configuration
including SECRET_KEY, database URIs, API tokens, and all other config values.

**3. File System Access**:
Through Python's built-in functions accessible via template injection, attackers can read
and write files on the server file system.

**4. Full Application Compromise**:
Since Jinja2 SSTI provides equivalent access to Python's exec(), the impact is identical
to arbitrary code execution -- complete server compromise.

VULNERABLE EXAMPLE:
```python
from flask import Flask, request, render_template_string

app = Flask(__name__)

@app.route('/hello')
def hello():
    name = request.args.get('name', 'World')
    # VULNERABLE: User input embedded in template string
    template = '<h1>Hello, ' + name + '!</h1>'
    return render_template_string(template)

# Attack: GET /hello?name={{config.SECRET_KEY}}
# Attack: GET /hello?name={{''.__class__.__mro__[1].__subclasses__()[408]('id',shell=True,stdout=-1).communicate()}}
```

SECURE EXAMPLE:
```python
from flask import Flask, request, render_template, render_template_string

app = Flask(__name__)

@app.route('/hello')
def hello():
    name = request.args.get('name', 'World')
    # SAFE Option 1: Use render_template with a file-based template
    return render_template('hello.html', name=name)

@app.route('/hello-inline')
def hello_inline():
    name = request.args.get('name', 'World')
    # SAFE Option 2: Pass user input as a template variable, never in the template string
    return render_template_string('<h1>Hello, {{ name }}!</h1>', name=name)
```

DETECTION AND PREVENTION:

**Pre-deployment checks**:
```bash
pathfinder scan --project . --ruleset cpf/python/PYTHON-FLASK-SEC-014
```

**Code Review Checklist**:
- [ ] No user input concatenated into render_template_string() first argument
- [ ] User data always passed as keyword arguments to template rendering functions
- [ ] render_template() with file-based templates preferred over render_template_string()
- [ ] Jinja2 sandbox enabled if dynamic template generation is required
- [ ] Template auto-escaping not disabled

COMPLIANCE:
- CWE-96: Improper Neutralization of Directives in Statically Saved Code
- OWASP Top 10 A03:2021 - Injection
- NIST SP 800-53 SI-10: Information Input Validation

REFERENCES:
- CWE-96: https://cwe.mitre.org/data/definitions/96.html
- Flask Jinja2 SSTI: https://nvisium.com/blog/2016/03/09/exploring-ssti-in-flask-jinja2.html
- Jinja2 SSTI Cheat Sheet: https://pequalsnp-team.github.io/cheatsheet/flask-jinja2-ssti
- Jinja2 Sandbox: https://jinja.palletsprojects.com/en/latest/sandbox/
- Flask render_template_string: https://flask.palletsprojects.com/en/latest/api/#flask.render_template_string

DETECTION SCOPE:
This rule performs inter-procedural taint analysis tracking data from Flask request sources
to render_template_string() sinks. No sanitizers are recognized because there is no safe
way to embed user input directly in a Jinja2 template string.
"""

from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets


class FlaskModule(QueryType):
    fqns = ["flask"]


@python_rule(
    id="PYTHON-FLASK-SEC-014",
    name="Flask Server-Side Template Injection",
    severity="CRITICAL",
    category="flask",
    cwe="CWE-96",
    tags="python,flask,ssti,template-injection,rce,owasp-a03,cwe-96",
    message="User input flows to render_template_string(). Use render_template() with separate template files.",
    owasp="A03:2021",
)
def detect_flask_ssti():
    """Detects Flask request data flowing to render_template_string()."""
    return flows(
        from_sources=[
            calls("request.args.get"),
            calls("request.form.get"),
            calls("request.values.get"),
            calls("request.get_json"),
        ],
        to_sinks=[
            FlaskModule.method("render_template_string").tracks(0),
            calls("render_template_string"),
        ],
        sanitized_by=[],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
