"""
PYTHON-DJANGO-001: Django SQL Injection in cursor.execute()

Security Impact: CRITICAL
CWE: CWE-89 (SQL Injection)
CVE: CVE-2022-34265 (Django SQL injection in Trunc/Extract)
OWASP: A03:2021 - Injection

DESCRIPTION:
This rule detects SQL injection vulnerabilities in Django applications where untrusted user input
flows directly into raw SQL execution via cursor.execute() or similar methods without proper
parameterization. This is one of the most critical web security vulnerabilities.

WHAT IS SQL INJECTION:

SQL injection occurs when an attacker can manipulate SQL queries by injecting malicious input.
In Django, this typically happens when developers bypass the ORM's built-in protections and
use raw SQL queries with string formatting or concatenation.

SECURITY IMPLICATIONS:

**1. Data Breach**:
Attackers can extract sensitive data from your database, including:
- User credentials (passwords, API keys)
- Personal information (emails, addresses, SSNs)
- Business data (financial records, trade secrets)

**2. Data Manipulation**:
Attackers can modify or delete data:
- Update admin privileges
- Delete critical records
- Modify financial transactions

**3. Authentication Bypass**:
Bypass login mechanisms entirely:
```python
# User input: ' OR '1'='1
username = request.GET.get('username')  # ' OR '1'='1
query = f"SELECT * FROM users WHERE username = '{username}'"
# Resulting query: SELECT * FROM users WHERE username = '' OR '1'='1'
# This returns ALL users, bypassing authentication
```

**4. Remote Code Execution** (in some databases):
- Execute system commands via xp_cmdshell (SQL Server)
- Read/write files via LOAD_FILE() (MySQL)
- Access OS through large objects (PostgreSQL)

VULNERABLE EXAMPLE:
```python
from django.db import connection
from django.http import HttpRequest, JsonResponse

def get_user_profile(request: HttpRequest):
    \"\"\"
    VULNERABLE: User input flows directly into SQL query.
    An attacker can inject: 1' OR '1'='1
    \"\"\"
    user_id = request.GET.get('user_id')  # Source: untrusted input

    cursor = connection.cursor()
    # DANGEROUS: f-string interpolation in SQL
    query = f"SELECT * FROM users WHERE id = {user_id}"
    cursor.execute(query)  # Sink: SQL execution

    result = cursor.fetchone()
    return JsonResponse({'user': result})

# Attack example:
# GET /profile?user_id=1' OR '1'='1 --
# Resulting query: SELECT * FROM users WHERE id = 1' OR '1'='1 --
# Returns all users instead of just one
```

SECURE EXAMPLE:
```python
from django.db import connection
from django.http import HttpRequest, JsonResponse

def get_user_profile(request: HttpRequest):
    \"\"\"
    SECURE: Uses parameterized queries with placeholders.
    \"\"\"
    user_id = request.GET.get('user_id')  # User input

    cursor = connection.cursor()
    # SAFE: Parameterized query with %s placeholder
    query = "SELECT * FROM users WHERE id = %s"
    cursor.execute(query, [user_id])  # Parameters passed separately

    result = cursor.fetchone()
    return JsonResponse({'user': result})

# Even with attack input, parameters are properly escaped:
# GET /profile?user_id=1' OR '1'='1
# Django escapes the input, query becomes:
# SELECT * FROM users WHERE id = '1\' OR \'1\'=\'1'
# No SQL injection possible
```

ALTERNATIVE SECURE APPROACHES:

**1. Use Django ORM** (Recommended):
```python
from django.contrib.auth.models import User

def get_user_profile(request):
    user_id = request.GET.get('user_id')
    # Django ORM automatically parameterizes queries
    user = User.objects.filter(id=user_id).first()
    return JsonResponse({'user': {'id': user.id, 'username': user.username}})
```

**2. Use Django's escape_sql()** (for complex cases):
```python
from django.db import connection
from django.db.backends.utils import escape_sql

def complex_query(request):
    search = request.GET.get('q')
    escaped = escape_sql(search)
    # Still use parameterized queries for values
    query = "SELECT * FROM products WHERE name LIKE %s"
    cursor.execute(query, [f'%{escaped}%'])
```

**3. Avoid .raw() and .extra()** (deprecated):
```python
# AVOID THIS (vulnerable if not careful):
User.objects.raw(f"SELECT * FROM users WHERE name = '{name}'")

# USE THIS INSTEAD:
User.objects.raw("SELECT * FROM users WHERE name = %s", [name])
```

DETECTION AND PREVENTION:

**Pre-deployment checks**:
```bash
# Scan Django project for SQL injection
pathfinder scan --project . --ruleset cpf/python/PYTHON-DJANGO-001

# Run static analysis in CI/CD
# .github/workflows/security.yml:
# - name: Scan for SQL injection
#   run: pathfinder ci --project . --ruleset cpf/python/django
```

**Code Review Checklist**:
- [ ] All cursor.execute() calls use parameterized queries (%s placeholders)
- [ ] No f-strings or .format() in SQL queries
- [ ] No string concatenation (+) in SQL queries
- [ ] ORM .raw() and .extra() methods use parameterization
- [ ] User input never directly interpolated into SQL

**Django Settings** (enable strict mode):
```python
# settings.py
DEBUG = False  # Never True in production

DATABASES = {
    'default': {
        # ... other settings
        'OPTIONS': {
            'sql_mode': 'STRICT_ALL_TABLES',  # MySQL/MariaDB
        }
    }
}
```

REAL-WORLD ATTACK SCENARIOS:

**1. Union-Based Injection**:
```python
# Attack: ?user_id=1 UNION SELECT username,password FROM admin_users--
# Attacker retrieves admin credentials
```

**2. Boolean-Based Blind Injection**:
```python
# Attack: ?user_id=1 AND 1=1  (returns results)
# Attack: ?user_id=1 AND 1=2  (returns nothing)
# Attacker infers database structure bit by bit
```

**3. Time-Based Blind Injection**:
```python
# Attack: ?user_id=1 AND SLEEP(5)
# If response is slow, injection successful
# Attacker extracts data one bit at a time
```

COMPLIANCE AND AUDITING:

**CIS Django Benchmark**:
> "All database queries must use parameterized statements"

**OWASP Top 10**:
SQL Injection is consistently ranked in Top 3 most critical web vulnerabilities

**PCI DSS Requirement 6.5.1**:
> "Injection flaws, particularly SQL injection"

**SOC 2 / ISO 27001**:
Requires input validation and parameterized queries

**GDPR Article 32**:
SQL injection can lead to massive data breaches subject to fines

MIGRATION GUIDE:

**Step 1: Identify all raw SQL usage**:
```bash
# Find all cursor.execute calls
grep -r "cursor.execute" --include="*.py"

# Find all .raw() usage
grep -r ".raw(" --include="*.py"
```

**Step 2: Replace with parameterized queries**:
```python
# BEFORE
query = f"SELECT * FROM users WHERE email = '{email}'"
cursor.execute(query)

# AFTER
query = "SELECT * FROM users WHERE email = %s"
cursor.execute(query, [email])
```

**Step 3: Test thoroughly**:
```python
# Test with SQL injection payloads
test_payloads = [
    "1' OR '1'='1",
    "1'; DROP TABLE users--",
    "1 UNION SELECT password FROM admin",
]

for payload in test_payloads:
    response = client.get(f'/api/user?id={payload}')
    # Should NOT return unauthorized data or cause errors
```

**Step 4: Add automated testing**:
```python
# tests/test_security.py
from django.test import TestCase

class SQLInjectionTests(TestCase):
    def test_user_profile_sql_injection(self):
        # Attempt SQL injection
        response = self.client.get("/profile?id=1' OR '1'='1")
        # Should not expose all users
        self.assertNotContains(response, "admin@")
```

FRAMEWORK-SPECIFIC NOTES:

**Django 4.2+**:
- QuerySet.extra() is deprecated (use .annotate() instead)
- QuerySet.raw() still supported but requires careful use

**Django REST Framework**:
```python
# Vulnerable
def get_queryset(self):
    user_id = self.request.query_params.get('id')
    return User.objects.raw(f"SELECT * FROM users WHERE id = {user_id}")

# Secure
def get_queryset(self):
    user_id = self.request.query_params.get('id')
    return User.objects.filter(id=user_id)  # Use ORM
```

REFERENCES:
- CWE-89: SQL Injection (https://cwe.mitre.org/data/definitions/89.html)
- CVE-2022-34265: Django SQL injection in Trunc/Extract
- OWASP A03:2021 - Injection (https://owasp.org/Top10/A03_2021-Injection/)
- Django Security Docs: https://docs.djangoproject.com/en/stable/topics/security/
- OWASP SQL Injection Prevention Cheat Sheet
- Bobby Tables: https://bobby-tables.com/python

DETECTION SCOPE:
This rule performs intra-procedural analysis only. It detects SQL injection when both the
source (user input) and sink (SQL execution) are in the same function. It will NOT detect
cases where user input is passed through multiple function calls before reaching SQL execution.

LIMITATION:
- Only detects flows within a single function (intra-procedural)
- Does not track dataflow across function boundaries (inter-procedural)
- May miss complex multi-function SQL injection patterns
"""

from rules.python_decorators import python_rule
from codepathfinder import calls, flows
from codepathfinder.presets import PropagationPresets


@python_rule(
    id="PYTHON-DJANGO-001",
    name="Django SQL Injection in cursor.execute()",
    severity="CRITICAL",
    category="django",
    cwe="CWE-89",
    cve="CVE-2022-34265",
    tags="python,django,sql-injection,orm,database,owasp-a03,cwe-89,parameterization,cursor,intra-procedural,critical,security",
    message="SQL injection vulnerability: User input flows to cursor.execute() without parameterization within a function. Use parameterized queries with %s placeholders.",
    owasp="A03:2021",
)
def detect_django_sql_injection():
    """
    Detects SQL injection where user input flows to Django cursor.execute() within a single function.

    LIMITATION: Only detects intra-procedural flows (within one function).
    Will NOT detect if request.GET is in one function and cursor.execute is in another.

    Example vulnerable code:
        user_id = request.GET.get('id')
        query = f"SELECT * FROM users WHERE id = {user_id}"
        cursor.execute(query)
    """
    return flows(
        from_sources=[
            calls("request.GET.get"),
            calls("request.POST.get"),
            calls("request.GET"),
            calls("request.POST"),
            calls("request.COOKIES"),
            calls("request.FILES"),
            calls("*.GET.get"),
            calls("*.POST.get"),
            calls("*.GET"),
            calls("*.POST"),
        ],
        to_sinks=[
            calls("execute"),
            calls("cursor.execute"),
            calls("*.execute"),
            calls("*.raw"),
            calls("*.extra"),
        ],
        sanitized_by=[
            calls("escape"),
            calls("escape_string"),
            calls("escape_sql"),
            calls("*.escape"),
            calls("*.escape_string"),
            calls("*.escape_sql"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="local",  # CRITICAL: Only intra-procedural analysis works
    )
