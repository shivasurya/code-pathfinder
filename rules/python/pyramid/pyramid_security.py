"""
Python Pyramid Security Rules

PYTHON-PYRAMID-SEC-001: CSRF Check Disabled Globally
PYTHON-PYRAMID-SEC-002: Direct Use of Response (XSS)
PYTHON-PYRAMID-SEC-003: SQLAlchemy SQL Injection

Security Impact: HIGH to CRITICAL
CWE: CWE-352 (Cross-Site Request Forgery),
     CWE-79 (Cross-Site Scripting),
     CWE-89 (SQL Injection)
OWASP: A03:2021 - Injection, A05:2021 - Security Misconfiguration

DESCRIPTION:
These rules detect common security vulnerabilities in Python Pyramid web applications,
covering three critical categories: CSRF protection bypass, Cross-Site Scripting (XSS)
via direct response construction, and SQL injection through SQLAlchemy raw queries.

Detected vulnerabilities:
- **CSRF disabled globally**: Calling `set_default_csrf_options(require_csrf=False)` disables
  CSRF protection for the entire application, exposing all state-changing endpoints to
  cross-site request forgery attacks
- **Direct Response XSS**: User input from request parameters flowing directly into
  `pyramid.response.Response()` without sanitization, enabling reflected XSS attacks
- **SQLAlchemy SQL injection**: User input from request parameters flowing into raw SQL
  operations via SQLAlchemy's `filter()`, `execute()`, `order_by()`, `group_by()`, or
  `having()` without parameterized queries

SECURITY IMPLICATIONS:

**1. CSRF (CWE-352)**:
With CSRF protection disabled, attackers can craft malicious pages that trigger state-changing
requests (transfers, password changes, data deletion) on behalf of authenticated users who
visit the attacker's page.

**2. Cross-Site Scripting (CWE-79)**:
When user input is embedded directly in HTTP responses without escaping, attackers can inject
JavaScript that executes in victims' browsers, stealing session cookies, performing actions
as the user, or redirecting to phishing sites.

**3. SQL Injection (CWE-89)**:
User input concatenated into SQL queries allows attackers to modify query logic, extract
sensitive data from the database, modify or delete records, or in some cases execute
operating system commands through the database.

VULNERABLE EXAMPLE:
```python
from pyramid.config import Configurator
from pyramid.response import Response


# SEC-001: CSRF disabled
config = Configurator()
config.set_default_csrf_options(require_csrf=False)


# SEC-002: Direct response XSS
def vulnerable_view(request):
    name = request.params.get('name')
    return Response(f"Hello {name}")


# SEC-003: SQLAlchemy SQL injection
def vulnerable_query(request):
    search = request.params.get('q')
    results = session.query(User).filter(f"name = '{search}'")
    return results


def vulnerable_order(request):
    sort_col = request.params.get('sort')
    results = session.query(Item).order_by(sort_col)
    return results
```

SECURE EXAMPLE:
```python
from pyramid.config import Configurator
from pyramid.response import Response
from markupsafe import escape
from sqlalchemy import text

# SECURE: CSRF protection enabled (default)
config = Configurator()
config.set_default_csrf_options(require_csrf=True, token='csrf_token')

# SECURE: Escape user input or use templates with auto-escaping
@view_config(route_name='greet', renderer='templates/greet.jinja2')
def greet(request):
    name = request.params.get('name', '')
    return {'name': name}  # Jinja2 auto-escapes by default

# Or if using Response directly:
@view_config(route_name='greet')
def greet_safe(request):
    name = escape(request.params.get('name', ''))
    return Response(f'<h1>Hello, {name}!</h1>')

# SECURE: Parameterized SQL queries
@view_config(route_name='search')
def search(request):
    query = request.params.get('q', '')
    results = DBSession.query(User).filter(
        User.name.like(f'%{query}%')  # ORM method - safe
    ).all()
    # Or with raw SQL using bindparams:
    stmt = text("SELECT * FROM users WHERE name LIKE :q").bindparams(q=f'%{query}%')
    return {'results': results}
```

DETECTION AND PREVENTION:
```bash
# Scan for Pyramid security issues
pathfinder scan --project . --ruleset cpf/python/PYTHON-PYRAMID-SEC-001

# CI/CD integration
- name: Check Pyramid security
  run: pathfinder ci --project . --ruleset cpf/python/pyramid
```

**Code Review Checklist**:
- [ ] CSRF protection is enabled globally (no require_csrf=False)
- [ ] Per-view CSRF exemptions are documented and justified
- [ ] All user input is escaped before inclusion in HTML responses
- [ ] Templates use auto-escaping (Jinja2, Chameleon with auto-escape)
- [ ] SQL queries use parameterized queries or ORM methods, never string formatting
- [ ] SQLAlchemy text() queries use bindparams() for user input

COMPLIANCE:
- OWASP A03:2021: Injection (SQL injection, XSS)
- OWASP A05:2021: Security Misconfiguration (CSRF disabled)
- CWE-352: Cross-Site Request Forgery
- CWE-79: Improper Neutralization of Input During Web Page Generation
- CWE-89: Improper Neutralization of Special Elements used in an SQL Command

REFERENCES:
- CWE-352: Cross-Site Request Forgery (https://cwe.mitre.org/data/definitions/352.html)
- CWE-79: Cross-Site Scripting (https://cwe.mitre.org/data/definitions/79.html)
- CWE-89: SQL Injection (https://cwe.mitre.org/data/definitions/89.html)
- Pyramid Security Documentation (https://docs.pylonsproject.org/projects/pyramid/en/latest/narr/security.html)
- OWASP CSRF Prevention Cheat Sheet (https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html)
- OWASP SQL Injection Prevention Cheat Sheet (https://cheatsheetseries.owasp.org/cheatsheets/SQL_Injection_Prevention_Cheat_Sheet.html)
"""

from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets


class PyramidConfigurator(QueryType):
    fqns = ["pyramid.config.Configurator"]


class PyramidResponse(QueryType):
    fqns = ["pyramid.response.Response", "pyramid.request.Response"]


class SQLAlchemyText(QueryType):
    fqns = ["sqlalchemy.text", "sqlalchemy.sql.text"]


_PYRAMID_SOURCES = [
    calls("request.params.get"),
    calls("request.params"),
    calls("request.GET.get"),
    calls("request.POST.get"),
    calls("request.matchdict.get"),
    calls("request.json_body.get"),
    calls("*.params.get"),
    calls("*.params"),
]


@python_rule(
    id="PYTHON-PYRAMID-SEC-001",
    name="Pyramid CSRF Check Disabled Globally",
    severity="HIGH",
    category="pyramid",
    cwe="CWE-352",
    tags="python,pyramid,csrf,security,owasp-a05,cwe-352",
    message="CSRF protection disabled globally via set_default_csrf_options(require_csrf=False).",
    owasp="A05:2021",
)
def detect_pyramid_csrf_disabled():
    """Detects Configurator.set_default_csrf_options() calls."""
    return calls("*.set_default_csrf_options")


@python_rule(
    id="PYTHON-PYRAMID-SEC-002",
    name="Pyramid Direct Response XSS",
    severity="HIGH",
    category="pyramid",
    cwe="CWE-79",
    tags="python,pyramid,xss,response,owasp-a03,cwe-79",
    message="User input flows directly to Response(). Use templates with auto-escaping.",
    owasp="A03:2021",
)
def detect_pyramid_response_xss():
    """Detects request data flowing to Pyramid Response."""
    return flows(
        from_sources=_PYRAMID_SOURCES,
        to_sinks=[
            PyramidResponse.method("__init__"),
            calls("Response"),
            calls("pyramid.response.Response"),
        ],
        sanitized_by=[
            calls("escape"),
            calls("markupsafe.escape"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="local",
    )


@python_rule(
    id="PYTHON-PYRAMID-SEC-003",
    name="Pyramid SQLAlchemy SQL Injection",
    severity="CRITICAL",
    category="pyramid",
    cwe="CWE-89",
    tags="python,pyramid,sqlalchemy,sql-injection,owasp-a03,cwe-89",
    message="User input flows to raw SQL in SQLAlchemy. Use parameterized queries with bindparams().",
    owasp="A03:2021",
)
def detect_pyramid_sqli():
    """Detects request data flowing to SQLAlchemy raw SQL."""
    return flows(
        from_sources=_PYRAMID_SOURCES,
        to_sinks=[
            calls("*.filter"),
            calls("*.order_by"),
            calls("*.group_by"),
            calls("*.having"),
            calls("*.execute"),
        ],
        sanitized_by=[
            calls("bindparams"),
            calls("*.bindparams"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="local",
    )
