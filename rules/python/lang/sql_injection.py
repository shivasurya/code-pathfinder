"""
SQL Injection Security Rules for Python Database Libraries

Rules in this file:
- PYTHON-LANG-SEC-080: psycopg2 SQL Injection (CWE-89)
- PYTHON-LANG-SEC-081: asyncpg SQL Injection (CWE-89)
- PYTHON-LANG-SEC-082: aiopg SQL Injection (CWE-89)
- PYTHON-LANG-SEC-083: pg8000 SQL Injection (CWE-89)
- PYTHON-LANG-SEC-084: Formatted SQL Query (CWE-89)

Security Impact: CRITICAL
CWE: CWE-89 (Improper Neutralization of Special Elements used in an SQL Command)
OWASP: A03:2021 - Injection

DESCRIPTION:
SQL injection occurs when user-controlled input is concatenated or interpolated
into SQL query strings without proper parameterization. Python database drivers
including psycopg2, asyncpg, aiopg, and pg8000 all support parameterized queries
that separate SQL structure from data values. When developers use string
formatting (f-strings, .format(), % operator, or concatenation) to build queries,
they create injection points that allow attackers to manipulate query logic.

SECURITY IMPLICATIONS:
A successful SQL injection attack can extract entire database contents (data
exfiltration), modify or delete records (data tampering), bypass authentication
checks, escalate privileges within the database, and in some configurations
execute operating system commands via database features like xp_cmdshell (MSSQL)
or COPY TO PROGRAM (PostgreSQL). Second-order SQL injection occurs when
previously stored data is used unsafely in subsequent queries. Blind SQL injection
techniques allow data extraction even when query results are not directly visible
to the attacker.

    # Attack scenario: authentication bypass
    username = request.form["username"]  # Attacker sends: ' OR '1'='1' --
    query = f"SELECT * FROM users WHERE username = '{username}' AND password = '{pwd}'"
    # Resulting query: SELECT * FROM users WHERE username = '' OR '1'='1' --' AND ...
    cursor.execute(query)  # Returns all users, bypassing auth

VULNERABLE EXAMPLE:
```python
import sqlite3

# SEC-080: psycopg2
import psycopg2
pg_conn = psycopg2.connect("dbname=test")
pg_cursor = pg_conn.cursor()
name = "user_input"
pg_cursor.execute("SELECT * FROM users WHERE name = '" + name + "'")

# SEC-081: asyncpg
import asyncpg
async def query_asyncpg():
    conn = await asyncpg.connect("postgresql://localhost/test")
    await conn.execute("SELECT * FROM users WHERE id = " + user_id)
    await conn.fetch("SELECT * FROM t WHERE x = " + val)

# SEC-084: formatted SQL (general)
conn2 = sqlite3.connect("test.db")
cursor = conn2.cursor()
cursor.execute("SELECT * FROM products WHERE id = " + product_id)
```

SECURE EXAMPLE:
```python
import psycopg2
# Parameterized query with psycopg2 (safe)
cursor.execute("SELECT * FROM users WHERE id = %s", (user_id,))
# Parameterized query with asyncpg (safe)
await conn.fetch("SELECT * FROM users WHERE id = $1", user_id)
# Use psycopg2.sql module for dynamic identifiers
from psycopg2 import sql
query = sql.SQL("SELECT * FROM {} WHERE id = %s").format(sql.Identifier(table_name))
cursor.execute(query, (user_id,))
```

DETECTION AND PREVENTION:
- Always use parameterized queries with placeholder syntax (%s, $1, ?)
- Never use f-strings, .format(), % operator, or + concatenation for SQL
- Use psycopg2.sql module for safe dynamic SQL composition (identifiers, literals)
- Apply ORM frameworks (SQLAlchemy, Django ORM) that handle parameterization
- Implement database user permissions with least-privilege principle
- Use Web Application Firewalls (WAF) as defense in depth

COMPLIANCE:
- CWE-89: Improper Neutralization of Special Elements used in an SQL Command
- OWASP A03:2021 - Injection
- SANS Top 25 (2023) - CWE-89: SQL Injection (ranked #3)
- PCI DSS v4.0 Requirement 6.2: Secure development practices
- NIST SP 800-53: SI-10 (Information Input Validation)

REFERENCES:
- https://cwe.mitre.org/data/definitions/89.html
- https://owasp.org/Top10/A03_2021-Injection/
- https://www.psycopg.org/docs/usage.html#passing-parameters-to-sql-queries
- https://magicstack.github.io/asyncpg/current/usage.html
- https://docs.python.org/3/library/sqlite3.html#sqlite3-placeholders
- https://cheatsheetseries.owasp.org/cheatsheets/SQL_Injection_Prevention_Cheat_Sheet.html
"""

from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets


class Psycopg2Cursor(QueryType):
    fqns = ["psycopg2.extensions.cursor", "psycopg2.extras.RealDictCursor"]
    patterns = ["*cursor*"]
    match_subclasses = True


class AsyncpgConnection(QueryType):
    fqns = ["asyncpg.Connection", "asyncpg.connection.Connection"]
    patterns = ["*Connection"]
    match_subclasses = True


class AiopgCursor(QueryType):
    fqns = ["aiopg.Cursor", "aiopg.cursor.Cursor"]
    patterns = ["*Cursor"]
    match_subclasses = True


class Pg8000Cursor(QueryType):
    fqns = ["pg8000.Cursor", "pg8000.core.Cursor"]
    patterns = ["*Cursor"]
    match_subclasses = True


@python_rule(
    id="PYTHON-LANG-SEC-080",
    name="psycopg2 SQL Injection",
    severity="CRITICAL",
    category="lang",
    cwe="CWE-89",
    tags="python,psycopg2,sql-injection,database,owasp-a03,cwe-89",
    message="String formatting in psycopg2 query. Use parameterized queries: cursor.execute(sql, params).",
    owasp="A03:2021",
)
def detect_psycopg2_sqli():
    """Detects potential SQL injection in psycopg2 cursor.execute()."""
    return Psycopg2Cursor.method("execute", "executemany")


@python_rule(
    id="PYTHON-LANG-SEC-081",
    name="asyncpg SQL Injection",
    severity="CRITICAL",
    category="lang",
    cwe="CWE-89",
    tags="python,asyncpg,sql-injection,database,cwe-89",
    message="String formatting in asyncpg query. Use parameterized queries with $1 placeholders.",
    owasp="A03:2021",
)
def detect_asyncpg_sqli():
    """Detects potential SQL injection in asyncpg connection methods."""
    return AsyncpgConnection.method("execute", "executemany", "fetch", "fetchrow", "fetchval")


@python_rule(
    id="PYTHON-LANG-SEC-082",
    name="aiopg SQL Injection",
    severity="CRITICAL",
    category="lang",
    cwe="CWE-89",
    tags="python,aiopg,sql-injection,database,cwe-89",
    message="String formatting in aiopg query. Use parameterized queries.",
    owasp="A03:2021",
)
def detect_aiopg_sqli():
    """Detects potential SQL injection in aiopg cursor.execute()."""
    return AiopgCursor.method("execute", "executemany")


@python_rule(
    id="PYTHON-LANG-SEC-083",
    name="pg8000 SQL Injection",
    severity="CRITICAL",
    category="lang",
    cwe="CWE-89",
    tags="python,pg8000,sql-injection,database,cwe-89",
    message="String formatting in pg8000 query. Use parameterized queries.",
    owasp="A03:2021",
)
def detect_pg8000_sqli():
    """Detects potential SQL injection in pg8000 cursor.execute()."""
    return Pg8000Cursor.method("execute", "executemany")


@python_rule(
    id="PYTHON-LANG-SEC-084",
    name="Formatted SQL Query",
    severity="HIGH",
    category="lang",
    cwe="CWE-89",
    tags="python,sql-injection,formatted-query,cwe-89",
    message="SQL query built with string formatting detected. Use parameterized queries instead.",
    owasp="A03:2021",
)
def detect_formatted_sql():
    """Detects cursor.execute() calls (audit for string formatting in SQL)."""
    return calls("cursor.execute", "cursor.executemany",
                 "connection.execute", "conn.execute")
