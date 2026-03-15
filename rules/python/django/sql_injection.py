"""
Python Django SQL Injection Rules

Rules:
- PYTHON-DJANGO-SEC-001: SQL Injection via cursor.execute() (CWE-89)
- PYTHON-DJANGO-SEC-002: SQL Injection via ORM .raw() (CWE-89)
- PYTHON-DJANGO-SEC-003: SQL Injection via ORM .extra() (CWE-89)
- PYTHON-DJANGO-SEC-004: SQL Injection via RawSQL Expression (CWE-89)
- PYTHON-DJANGO-SEC-005: Raw SQL Usage Detected (CWE-89, Audit)
- PYTHON-DJANGO-SEC-006: Tainted SQL String Construction (CWE-89)

Security Impact: CRITICAL
CWE: CWE-89 (Improper Neutralization of Special Elements used in an SQL Command)
OWASP: A03:2021 - Injection

DESCRIPTION:
These rules detect SQL injection vulnerabilities in Django applications where untrusted
user input from HTTP requests flows into raw SQL queries without proper parameterization.
Django's ORM provides safe query construction by default, but developers sometimes bypass
this protection by using raw SQL via cursor.execute(), Model.objects.raw(), QuerySet.extra(),
or RawSQL expressions. When user input reaches these functions without parameterization,
attackers can manipulate SQL queries to read, modify, or delete arbitrary database records.

SECURITY IMPLICATIONS:

**1. Data Breach**:
An attacker can extract sensitive data from any table in the database using UNION-based
or blind SQL injection techniques, bypassing all application-level access controls.

**2. Authentication Bypass**:
SQL injection in login queries allows attackers to bypass authentication entirely
by injecting conditions that always evaluate to true (e.g., ' OR 1=1 --).

**3. Data Manipulation**:
Attackers can INSERT, UPDATE, or DELETE records, corrupt data integrity, or escalate
privileges by modifying user roles directly in the database.

**4. Remote Code Execution**:
On certain database backends (e.g., PostgreSQL with COPY, MySQL with INTO OUTFILE),
SQL injection can lead to file system access or command execution on the database server.

VULNERABLE EXAMPLE:
```python
from django.db import connection
from django.db.models.expressions import RawSQL
from django.http import HttpRequest


# SEC-001: cursor.execute with request data
def vulnerable_cursor(request):
    user_id = request.GET.get('id')
    cursor = connection.cursor()
    query = f"SELECT * FROM users WHERE id = {user_id}"
    cursor.execute(query)
    return cursor.fetchone()


# SEC-002: ORM .raw() with request data
def vulnerable_raw(request):
    name = request.POST.get('name')
    users = User.objects.raw(f"SELECT * FROM users WHERE name = '{name}'")
    return users


# SEC-003: ORM .extra() with request data
def vulnerable_extra(request):
    where_clause = request.GET.get('filter')
    results = Article.objects.extra(where=[where_clause])
    return results


# SEC-004: RawSQL expression with request data
def vulnerable_rawsql(request):
    order = request.GET.get('order')
    expr = RawSQL(f"SELECT * FROM products ORDER BY {order}", [])
    return expr


# SEC-005: Raw SQL usage (audit)
def audit_rawsql():
    expr = RawSQL("SELECT 1", [])
    return expr


# SEC-006: Tainted SQL string (same as SEC-001 pattern)
def vulnerable_tainted_sql(request):
    search = request.GET.get('q')
    query = "SELECT * FROM items WHERE name LIKE '%" + search + "%'"
    cursor = connection.cursor()
    cursor.execute(query)
    return cursor.fetchall()
```

SECURE EXAMPLE:
```python
from django.http import JsonResponse
from django.db import connection

def search_users(request):
    # SECURE: Parameterized query with %s placeholder
    username = request.GET.get('username')
    cursor = connection.cursor()
    cursor.execute("SELECT * FROM users WHERE username = %s", [username])
    results = cursor.fetchall()
    return JsonResponse({'users': results})

def get_orders(request):
    # SECURE: Use Django ORM with safe filtering
    status = request.GET.get('status')
    orders = Order.objects.filter(status=status).values()
    return JsonResponse({'orders': list(orders)})

def get_orders_raw(request):
    # SECURE: Parameterized .raw() query
    status = request.GET.get('status')
    orders = Order.objects.raw("SELECT * FROM orders WHERE status = %s", [status])
    return JsonResponse({'orders': list(orders)})
```

DETECTION AND PREVENTION:

**Key Mitigation Strategies**:
- Always use parameterized queries with %s placeholders in cursor.execute()
- Prefer Django ORM methods (filter, exclude, annotate) over raw SQL
- When .raw() is necessary, always pass parameters as a separate list
- Replace .extra() with .annotate() and F()/Q() expressions
- Never use string formatting (%, .format(), f-strings) to build SQL queries
- Use Django's connection.cursor() with proper parameterization

**Pre-deployment checks**:
```bash
pathfinder scan --project . --ruleset cpf/python/django/sql-injection
```

COMPLIANCE:
- CWE-89: Improper Neutralization of Special Elements used in an SQL Command
- OWASP A03:2021 - Injection
- SANS Top 25: CWE-89 ranked #3
- NIST SP 800-53: SI-10 (Information Input Validation)
- PCI DSS Requirement 6.5.1: Injection flaws

REFERENCES:
- CWE-89: https://cwe.mitre.org/data/definitions/89.html
- OWASP SQL Injection: https://owasp.org/www-community/attacks/SQL_Injection
- Django Raw SQL: https://docs.djangoproject.com/en/stable/topics/db/sql/
- Django Security: https://docs.djangoproject.com/en/stable/topics/security/#sql-injection-protection
"""

from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets


class DBCursor(QueryType):
    fqns = ["sqlite3.Cursor", "mysql.connector.cursor.MySQLCursor",
            "psycopg2.extensions.cursor", "pymysql.cursors.Cursor"]
    patterns = ["*Cursor"]
    match_subclasses = True


class DjangoORM(QueryType):
    fqns = ["django.db.models.Manager", "django.db.models.QuerySet"]
    patterns = ["*Manager", "*QuerySet"]
    match_subclasses = True


class DjangoExpressions(QueryType):
    fqns = ["django.db.models.expressions"]


# Common Django request sources
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
    id="PYTHON-DJANGO-SEC-001",
    name="Django SQL Injection via cursor.execute()",
    severity="CRITICAL",
    category="django",
    cwe="CWE-89",
    tags="python,django,sql-injection,cursor,owasp-a03,cwe-89",
    message="User input flows to cursor.execute() without parameterization. Use %s placeholders.",
    owasp="A03:2021",
)
def detect_django_cursor_sqli():
    """Detects request data flowing to cursor.execute()."""
    return flows(
        from_sources=_DJANGO_SOURCES,
        to_sinks=[
            DBCursor.method("execute", "executemany").tracks(0),
            calls("cursor.execute"),
            calls("*.execute"),
        ],
        sanitized_by=[
            calls("escape"),
            calls("escape_string"),
            calls("escape_sql"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )


@python_rule(
    id="PYTHON-DJANGO-SEC-002",
    name="Django SQL Injection via ORM .raw()",
    severity="CRITICAL",
    category="django",
    cwe="CWE-89",
    tags="python,django,sql-injection,orm-raw,owasp-a03,cwe-89",
    message="User input flows to .raw() query. Use parameterized .raw() with %s placeholders.",
    owasp="A03:2021",
)
def detect_django_raw_sqli():
    """Detects request data flowing to Model.objects.raw()."""
    return flows(
        from_sources=_DJANGO_SOURCES,
        to_sinks=[
            DjangoORM.method("raw").tracks(0),
            calls("*.objects.raw"),
            calls("*.raw"),
        ],
        sanitized_by=[
            calls("escape"),
            calls("escape_string"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )


@python_rule(
    id="PYTHON-DJANGO-SEC-003",
    name="Django SQL Injection via ORM .extra()",
    severity="HIGH",
    category="django",
    cwe="CWE-89",
    tags="python,django,sql-injection,orm-extra,owasp-a03,cwe-89",
    message="User input flows to .extra() query. Use .annotate() or parameterized queries instead.",
    owasp="A03:2021",
)
def detect_django_extra_sqli():
    """Detects request data flowing to QuerySet.extra()."""
    return flows(
        from_sources=_DJANGO_SOURCES,
        to_sinks=[
            DjangoORM.method("extra").tracks(0),
            calls("*.objects.extra"),
            calls("*.extra"),
        ],
        sanitized_by=[
            calls("escape"),
            calls("escape_string"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )


@python_rule(
    id="PYTHON-DJANGO-SEC-004",
    name="Django SQL Injection via RawSQL Expression",
    severity="CRITICAL",
    category="django",
    cwe="CWE-89",
    tags="python,django,sql-injection,rawsql,owasp-a03,cwe-89",
    message="User input flows to RawSQL() expression. Use parameterized queries.",
    owasp="A03:2021",
)
def detect_django_rawsql_sqli():
    """Detects request data flowing to RawSQL()."""
    return flows(
        from_sources=_DJANGO_SOURCES,
        to_sinks=[
            DjangoExpressions.method("RawSQL").tracks(0),
            calls("RawSQL"),
            calls("*.RawSQL"),
        ],
        sanitized_by=[
            calls("escape"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )


@python_rule(
    id="PYTHON-DJANGO-SEC-005",
    name="Raw SQL Usage Detected (Audit)",
    severity="MEDIUM",
    category="django",
    cwe="CWE-89",
    tags="python,django,sql-injection,raw-sql,audit,cwe-89",
    message="Raw SQL usage detected. Ensure parameterized queries are used.",
    owasp="A03:2021",
)
def detect_django_raw_sql_audit():
    """Audit rule: detects any usage of raw SQL APIs."""
    return DjangoExpressions.method("RawSQL")


@python_rule(
    id="PYTHON-DJANGO-SEC-006",
    name="Tainted SQL String Construction",
    severity="HIGH",
    category="django",
    cwe="CWE-89",
    tags="python,django,sql-injection,string-format,owasp-a03,cwe-89",
    message="User input used in SQL string construction. Use parameterized queries.",
    owasp="A03:2021",
)
def detect_django_tainted_sql_string():
    """Detects request data used in string formatting that reaches execute()."""
    return flows(
        from_sources=_DJANGO_SOURCES,
        to_sinks=[
            calls("cursor.execute"),
            calls("*.execute"),
        ],
        sanitized_by=[
            calls("escape"),
            calls("escape_string"),
            calls("escape_sql"),
            calls("int"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
