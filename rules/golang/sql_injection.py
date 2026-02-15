"""
GO-SEC-001: SQL Injection via Unsanitized Input

VULNERABILITY DESCRIPTION:
SQL injection occurs when user-controlled input is used in database queries without
proper sanitization or parameterization. In Go, this typically happens when concatenating
user input into SQL queries before passing to database/sql functions.

SEVERITY: CRITICAL
CWE: CWE-89 (SQL Injection)
OWASP: A03:2021 (Injection)

IMPACT:
- Data breach (unauthorized access to sensitive data)
- Data manipulation (UPDATE/DELETE statements)
- Authentication bypass
- Remote code execution (in some database configurations)

VULNERABLE PATTERNS:
- HTTP request parameters flowing to database/sql.DB.Query()
- URL query parameters used in SQL strings
- Form values concatenated into SQL statements
- Gin framework parameters used in database queries

SECURE PATTERNS:
- Use parameterized queries: db.Query("SELECT * FROM users WHERE id = $1", userID)
- Use ORM query builders with parameter binding
- Validate and sanitize input before use
- Use prepared statements

REFERENCES:
- https://owasp.org/www-community/attacks/SQL_Injection
- https://cwe.mitre.org/data/definitions/89.html
- https://cheatsheetseries.owasp.org/cheatsheets/SQL_Injection_Prevention_Cheat_Sheet.html
"""

from codepathfinder import rule, calls

@rule(
    id="GO-SEC-001",
    severity="CRITICAL",
    cwe="CWE-89",
    owasp="A03:2021"
)
def go_sql_injection():
    """Detects potential SQL injection in Go database calls.
    Flags calls to database query methods that may execute SQL."""
    return calls("*Query", "*Exec", "*QueryRow")
