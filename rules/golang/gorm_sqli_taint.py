"""
GO-GORM-SQLI Rules: SQL injection via GORM ORM.

GO-GORM-SQLI-001: SQL injection via GORM Raw/Exec (raw SQL methods)
GO-GORM-SQLI-002: SQL injection via GORM query builder methods (Order, Group, Having,
                  Distinct, Select, Pluck, Where) with string concatenation

Security Impact: CRITICAL
CWE: CWE-89 (SQL Injection)
OWASP: A03:2021 — Injection

DESCRIPTION:
GORM's Raw() and Exec() methods accept raw SQL strings — when user-controlled
input flows into these methods without parameterization, it creates SQL injection.

Additionally, GORM's query builder methods (Order, Group, Having, Where, Select, Pluck)
accept raw SQL strings for clauses. These are also vulnerable when user input is
concatenated directly: db.Order("name " + direction) — direction controls ORDER BY
direction, enabling injection into the SQL query structure.

VULNERABLE EXAMPLES:
    // Raw SQL injection
    search := c.Query("search")
    db.Raw("SELECT * FROM users WHERE name = '" + search + "'")

    // ORDER BY injection
    sort := c.Query("sort")
    db.Order(sort).Find(&users)  // attacker controls sort direction/column

    // WHERE injection
    filter := c.Query("filter")
    db.Where(filter).Find(&users)  // raw WHERE clause from user

SECURE EXAMPLES:
    // GORM parameterized query
    db.Raw("SELECT * FROM users WHERE name = ?", search)

    // GORM query builder (safe — column and direction validated)
    if sort == "asc" || sort == "desc" {
        db.Order("name " + sort).Find(&users)
    }

REFERENCES:
- CWE-89: https://cwe.mitre.org/data/definitions/89.html
- GORM SQL Builder: https://gorm.io/docs/sql_builder.html
- GORM Security: https://gorm.io/docs/security.html
"""

from codepathfinder.go_rule import (
    GoGinContext,
    GoEchoContext,
    GoFiberCtx,
    GoGormDB,
    GoHTTPRequest,
    GoStrconv,
    QueryType,
)
from codepathfinder import flows
from codepathfinder.presets import PropagationPresets
from codepathfinder.go_decorators import go_rule


class GoChiRouter(QueryType):
    """github.com/go-chi/chi — Chi HTTP router URL parameters."""

    fqns = ["github.com/go-chi/chi/v5"]
    patterns = ["chi.*"]
    match_subclasses = False


@go_rule(
    id="GO-GORM-SQLI-001",
    severity="CRITICAL",
    cwe="CWE-89",
    owasp="A03:2021",
    tags="go,security,sql-injection,gorm,CWE-89,OWASP-A03",
    message=(
        "User-controlled input flows into GORM Raw() or Exec() with raw SQL. "
        "This creates a SQL injection vulnerability — attackers can modify the query. "
        "Use GORM parameterized queries: db.Raw('SELECT * WHERE name = ?', name)"
    ),
)
def detect_gorm_sqli():
    """Detect SQL injection via GORM Raw/Exec with user-controlled input.

    GORM's Raw() and Exec() accept raw SQL strings. When user input flows into
    these methods without parameterization, it creates SQL injection.

    Bad:  db.Raw("SELECT * FROM users WHERE id = " + userID)
    Good: db.Raw("SELECT * FROM users WHERE id = ?", userID)
    """
    return flows(
        from_sources=[
            GoGinContext.method(
                "Query", "Param", "PostForm", "GetHeader",
                "ShouldBindJSON", "BindJSON", "GetRawData"
            ),
            GoEchoContext.method("QueryParam", "FormValue", "Param", "PathParam"),
            GoFiberCtx.method("Params", "Query", "FormValue", "Get"),
            GoHTTPRequest.method("FormValue", "PostFormValue"),
            GoHTTPRequest.attr("Body", "URL.Path", "URL.RawQuery"),
        ],
        to_sinks=[
            GoGormDB.method("Raw", "Exec"),
        ],
        sanitized_by=[
            GoStrconv.method("Atoi", "ParseInt", "ParseFloat", "ParseUint"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )


@go_rule(
    id="GO-GORM-SQLI-002",
    severity="HIGH",
    cwe="CWE-89",
    owasp="A03:2021",
    tags="go,security,sql-injection,gorm,query-builder,CWE-89,OWASP-A03",
    message=(
        "User-controlled input flows into GORM query builder methods (Order, Group, Having, "
        "Where, Distinct, Select, Pluck) as raw SQL strings. "
        "Attackers can inject SQL via ORDER BY, GROUP BY, or WHERE clauses. "
        "Validate user input against an allowlist of permitted column names and sort directions "
        "before passing to GORM builder methods."
    ),
)
def detect_gorm_query_builder_sqli():
    """Detect SQL injection via GORM query builder methods with user input.

    GORM's Order, Group, Having, Where, Select, Distinct, Pluck all accept
    raw SQL strings for query clauses. User input injected here can modify
    the query structure (column names, sort direction, filter conditions).

    Bad:  db.Order(c.Query("sort")).Find(&results)
    Good: if sort == "asc" || sort == "desc" { db.Order("name " + sort)... }
    """
    return flows(
        from_sources=[
            GoGinContext.method(
                "Query", "Param", "PostForm", "GetHeader",
                "ShouldBindJSON", "BindJSON", "GetRawData"
            ),
            GoEchoContext.method("QueryParam", "FormValue", "Param", "PathParam"),
            GoFiberCtx.method("Params", "Query", "FormValue", "Get"),
            GoHTTPRequest.method("FormValue", "PostFormValue"),
            GoHTTPRequest.attr("URL.RawQuery", "URL.Path"),
            GoChiRouter.method("URLParam"),
        ],
        to_sinks=[
            GoGormDB.method(
                "Order", "Group", "Having", "Where",
                "Distinct", "Select", "Pluck",
                "Not", "Or", "Joins",
            ),
        ],
        sanitized_by=[
            GoStrconv.method("Atoi", "ParseInt", "ParseFloat", "ParseUint"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
