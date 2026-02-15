"""
GO-SEC-001: SQL Injection via Unsanitized Input

Security Impact: CRITICAL
CWE: CWE-89 (SQL Injection)
OWASP: A03:2021 (Injection)

DESCRIPTION:
This rule detects calls to database/sql query methods (Query, Exec, QueryRow) that
may execute SQL statements. SQL injection occurs when user-controlled input is
concatenated into SQL queries without proper sanitization or parameterization,
allowing attackers to manipulate the query logic.

SECURITY IMPLICATIONS:
SQL injection is one of the most dangerous web application vulnerabilities, allowing
attackers to:

1. **Data Breach**: Access sensitive data including passwords, credit cards, PII
2. **Authentication Bypass**: Login as any user without knowing their password
3. **Data Manipulation**: Modify or delete database records (UPDATE/DELETE)
4. **Privilege Escalation**: Grant administrative privileges to attacker accounts
5. **Remote Code Execution**: Execute OS commands via xp_cmdshell (SQL Server) or similar
6. **Database Enumeration**: Extract database schema, table names, column names

VULNERABLE EXAMPLE:
```go
func getUser(w http.ResponseWriter, r *http.Request) {
    db, _ := sql.Open("postgres", "...")

    // CRITICAL: SQL injection vulnerability
    userID := r.FormValue("id")
    query := "SELECT * FROM users WHERE id = '" + userID + "'"
    rows, _ := db.Query(query)

    // Attack: ?id=1' OR '1'='1
    // Executes: SELECT * FROM users WHERE id = '1' OR '1'='1'
    // Returns ALL users instead of one
}
```

SECURE EXAMPLE:
```go
func getUser(w http.ResponseWriter, r *http.Request) {
    db, _ := sql.Open("postgres", "...")

    // SECURE: Use parameterized query
    userID := r.FormValue("id")
    query := "SELECT * FROM users WHERE id = $1"
    rows, err := db.Query(query, userID)
    if err != nil {
        http.Error(w, "Query failed", 500)
        return
    }
    defer rows.Close()

    // User input is safely parameterized
    // Postgres treats it as data, not SQL code
}
```

BEST PRACTICES:
1. **Always use parameterized queries**: Use $1, $2 placeholders (Postgres) or ? (MySQL)
2. **Never concatenate user input**: Avoid building SQL strings with + operator
3. **Use prepared statements**: db.Prepare() for repeated queries
4. **Use ORM libraries**: GORM, sqlx with parameter binding
5. **Input validation**: Validate data types (e.g., parse integers before querying)
6. **Principle of least privilege**: Database user should have minimal permissions

PARAMETER SYNTAX BY DATABASE:
```go
// PostgreSQL
db.Query("SELECT * FROM users WHERE id = $1 AND role = $2", id, role)

// MySQL
db.Query("SELECT * FROM users WHERE id = ? AND role = ?", id, role)

// SQLite
db.Query("SELECT * FROM users WHERE id = ? AND role = ?", id, role)
```

DETECTION LIMITATIONS:
This rule uses pattern matching and flags ALL calls to Query, Exec, and QueryRow
methods. It cannot determine if:
- The query uses parameterized statements correctly
- The input is actually user-controlled

Manual review is required to verify if detected calls are vulnerable.

REMEDIATION:
1. Replace string concatenation with parameterized queries
2. Use db.Prepare() for prepared statements
3. Consider using ORM libraries (GORM) or query builders (sqlx)
4. Add input validation and type checking
5. Review all database queries for proper parameter binding

REFERENCES:
- OWASP SQL Injection: https://owasp.org/www-community/attacks/SQL_Injection
- CWE-89: SQL Injection: https://cwe.mitre.org/data/definitions/89.html
- OWASP SQL Injection Prevention Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/SQL_Injection_Prevention_Cheat_Sheet.html
- Go database/sql documentation: https://pkg.go.dev/database/sql
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
