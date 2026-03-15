"""
PYTHON-FLASK-SEC-003: Flask SQL Injection via Tainted String

Security Impact: CRITICAL
CWE: CWE-89 (Improper Neutralization of Special Elements used in an SQL Command)
OWASP: A03:2021 - Injection

DESCRIPTION:
This rule detects SQL injection vulnerabilities in Flask applications where user-controlled
input from HTTP request parameters flows into raw SQL query execution without parameterized
queries. When user input is concatenated or interpolated directly into SQL strings and passed
to database cursor execute methods, attackers can manipulate query logic to extract, modify,
or destroy data.

SQL injection remains one of the most prevalent and dangerous web application vulnerabilities.
In Flask applications, this commonly occurs when developers construct SQL queries using string
formatting with data obtained from request.args, request.form, or request.get_json(), then
pass the resulting string to cursor.execute().

SECURITY IMPLICATIONS:

**1. Data Exfiltration**:
Attackers can use UNION-based injection, blind injection, or out-of-band techniques to
extract entire database contents including credentials, personal data, and business secrets.

**2. Authentication Bypass**:
Login forms vulnerable to SQL injection allow attackers to bypass authentication entirely
by injecting conditions that always evaluate to true (e.g., `' OR 1=1 --`).

**3. Data Manipulation**:
INSERT, UPDATE, and DELETE statements can be injected to modify or destroy records, corrupt
referential integrity, or plant backdoor accounts.

**4. Remote Code Execution**:
On certain database systems (e.g., PostgreSQL with COPY, MySQL with INTO OUTFILE, MSSQL
with xp_cmdshell), SQL injection can escalate to operating system command execution.

VULNERABLE EXAMPLE:
```python
# --- file: app.py ---
from flask import Flask, request
from db import query_user

app = Flask(__name__)

@app.route('/user')
def get_user():
    username = request.args.get('username')
    result = query_user(username)  # Tainted data crosses file boundary
    return str(result)

# --- file: db.py ---
import sqlite3

def get_connection():
    return sqlite3.connect('app.db')

def query_user(name):
    conn = get_connection()
    cursor = conn.cursor()
    # VULNERABLE: Tainted 'name' from app.py flows into raw SQL
    cursor.execute("SELECT * FROM users WHERE name = '" + name + "'")
    return cursor.fetchall()

# Attack: GET /user?username=' OR 1=1 --
# Taint flows: request.args.get() → query_user(username) → cursor.execute(name)
```

SECURE EXAMPLE:
```python
from flask import Flask, request
import sqlite3

app = Flask(__name__)

@app.route('/users/search')
def search_users():
    username = request.args.get('username')
    conn = sqlite3.connect('app.db')
    cursor = conn.cursor()
    # SAFE: Parameterized query prevents injection
    cursor.execute("SELECT * FROM users WHERE name = ?", (username,))
    return {'results': cursor.fetchall()}

# Alternative: Use an ORM like SQLAlchemy
from sqlalchemy import select
result = db.session.execute(select(User).filter_by(name=username))
```

DETECTION AND PREVENTION:

**Pre-deployment checks**:
```bash
# Scan for SQL injection vulnerabilities
pathfinder scan --project . --ruleset cpf/python/PYTHON-FLASK-SEC-003
```

**Code Review Checklist**:
- [ ] No string concatenation or f-strings in SQL queries with user input
- [ ] All database queries use parameterized placeholders (?, %s, :param)
- [ ] ORM queries use filter methods rather than raw SQL where possible
- [ ] Input validation applied before database operations

COMPLIANCE:
- CWE-89: Improper Neutralization of Special Elements used in an SQL Command
- OWASP Top 10 A03:2021 - Injection
- SANS Top 25 (CWE-89 ranked #3)
- PCI DSS Requirement 6.5.1: Injection Flaws

REFERENCES:
- CWE-89: https://cwe.mitre.org/data/definitions/89.html
- OWASP SQL Injection: https://owasp.org/www-community/attacks/SQL_Injection
- OWASP SQL Injection Prevention Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/SQL_Injection_Prevention_Cheat_Sheet.html
- SQLAlchemy Parameterized Queries: https://docs.sqlalchemy.org/en/14/core/tutorial.html#using-textual-sql

DETECTION SCOPE:
This rule performs inter-procedural taint analysis tracking data from Flask request sources
(request.args.get, request.form.get, etc.) to database cursor execute/executemany sinks.
Recognized sanitizers include escape() and escape_string() functions.
"""

from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets


class DBCursor(QueryType):
    fqns = ["sqlite3.Cursor", "mysql.connector.cursor.MySQLCursor",
            "psycopg2.extensions.cursor", "pymysql.cursors.Cursor"]
    patterns = ["*Cursor"]
    match_subclasses = True


@python_rule(
    id="PYTHON-FLASK-SEC-003",
    name="Flask SQL Injection via Tainted String",
    severity="CRITICAL",
    category="flask",
    cwe="CWE-89",
    tags="python,flask,sql-injection,database,owasp-a03,cwe-89",
    message="User input flows to SQL execution without parameterization. Use parameterized queries.",
    owasp="A03:2021",
)
def detect_flask_sql_injection():
    """Detects Flask request data flowing to SQL execution."""
    return flows(
        from_sources=[
            calls("request.args.get"),
            calls("request.form.get"),
            calls("request.values.get"),
            calls("request.get_json"),
            calls("request.cookies.get"),
            calls("request.headers.get"),
        ],
        to_sinks=[
            DBCursor.method("execute", "executemany").tracks(0),
        ],
        sanitized_by=[
            calls("escape"),
            calls("escape_string"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
