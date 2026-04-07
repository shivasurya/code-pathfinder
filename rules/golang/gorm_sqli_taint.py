"""
GO-GORM-SQLI-001: SQL Injection via GORM Raw/Exec with Unsanitized Input

Security Impact: CRITICAL
CWE: CWE-89 (SQL Injection)
OWASP: A03:2021 (Injection)

DESCRIPTION:
GORM's Raw() and Exec() methods accept raw SQL strings. When user-controlled
input from HTTP request parameters flows into these methods without proper
parameterization, it creates a SQL injection vulnerability.

VULNERABLE EXAMPLE:
```go
func searchHandler(c *gin.Context) {
    search := c.Query("search")
    db.Raw("SELECT * FROM users WHERE name = '" + search + "'")
}
```

SECURE EXAMPLE:
```go
func searchHandler(c *gin.Context) {
    search := c.Query("search")
    // SECURE: Use GORM parameterized query
    db.Raw("SELECT * FROM users WHERE name = ?", search)
}
```

BEST PRACTICES:
1. Always use GORM parameterized queries with ? placeholders
2. Never concatenate user input directly into SQL strings
3. Use GORM's query builder methods (Where, Find, First) with struct binding
4. Validate and sanitize user input at entry points

REFERENCES:
- CWE-89: https://cwe.mitre.org/data/definitions/89.html
- OWASP SQL Injection: https://owasp.org/www-community/attacks/SQL_Injection
- GORM Raw SQL: https://gorm.io/docs/sql_builder.html
"""

from codepathfinder.go_rule import (
    GoGinContext,
    GoEchoContext,
    GoGormDB,
    GoHTTPRequest,
    GoStrconv,
)
from codepathfinder import flows
from codepathfinder.presets import PropagationPresets
from rules.go_decorators import go_rule


@go_rule(id="GO-GORM-SQLI-001", severity="CRITICAL", cwe="CWE-89", owasp="A03:2021")
def detect_gorm_sqli():
    """Detect SQL injection via GORM Raw/Exec with user-controlled input.

    GORM's Raw() and Exec() methods accept raw SQL strings. When user input
    flows into these methods without parameterization, it creates a SQL
    injection vulnerability.

    Good: db.Raw("SELECT * FROM users WHERE id = ?", userID)
    Bad:  db.Raw("SELECT * FROM users WHERE id = " + userID)
    """
    return flows(
        from_sources=[
            GoGinContext.method("Query", "Param", "PostForm", "GetHeader"),
            GoEchoContext.method("QueryParam", "FormValue", "Param"),
            GoHTTPRequest.method("FormValue"),
        ],
        to_sinks=[GoGormDB.method("Raw", "Exec")],
        sanitized_by=[GoStrconv.method("Atoi", "ParseInt", "ParseFloat")],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
