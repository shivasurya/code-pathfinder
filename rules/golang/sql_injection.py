"""
SQL Injection rules for Go — covering database/sql, pgx, sqlx drivers.

GO-SEC-001: SQL injection via database/sql (standard library DB)
GO-SQLI-002: SQL injection via pgx (PostgreSQL native driver)
GO-SQLI-003: SQL injection via sqlx (extended database/sql)

Sources: gin.Context, echo.Context, fiber.Ctx, net/http.Request, gorilla/mux, chi
Sinks:   database/sql, pgx, sqlx — all query/exec method variants

L1: QueryType both sides — type-inferred sources and sinks.
scope="global" — inter-procedural cross-file taint analysis.

REFERENCES:
- CWE-89: https://cwe.mitre.org/data/definitions/89.html
- OWASP SQL Injection: https://owasp.org/www-community/attacks/SQL_Injection
"""

from codepathfinder.go_rule import (
    GoGinContext,
    GoHTTPRequest,
    GoSQLDB,
    GoPgxConn,
    GoSqlxDB,
    GoEchoContext,
    GoFiberCtx,
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


class GoGorillaMux(QueryType):
    """github.com/gorilla/mux — Gorilla mux URL variables."""

    fqns = ["github.com/gorilla/mux"]
    patterns = ["mux.*"]
    match_subclasses = False


@go_rule(
    id="GO-SEC-001",
    severity="CRITICAL",
    cwe="CWE-89",
    owasp="A03:2021",
    tags="go,security,sql-injection,database,CWE-89,OWASP-A03",
    message=(
        "User-controlled input flows into a database/sql query method (Query, Exec, QueryRow). "
        "This creates a SQL injection vulnerability — attackers can modify the SQL statement "
        "to bypass authentication, exfiltrate data, or destroy the database. "
        "Use parameterized queries: db.Query('SELECT * FROM users WHERE id = $1', userID)"
    ),
)
def go_sql_injection():
    """HTTP/Gin/Echo/Fiber request input reaches database/sql query methods — SQL injection.

    L1: QueryType both sides (GoHTTPRequest/GoGinContext/GoEchoContext → GoSQLDB).
    scope=global — inter-procedural cross-file taint.

    Bad:  db.Query("SELECT * FROM users WHERE id = " + userID)
    Good: db.Query("SELECT * FROM users WHERE id = $1", userID)
    """
    return flows(
        from_sources=[
            GoGinContext.method(
                "Param", "Query", "PostForm", "GetRawData",
                "ShouldBindJSON", "BindJSON", "GetHeader"
            ),
            GoEchoContext.method("QueryParam", "FormValue", "Param", "PathParam"),
            GoFiberCtx.method("Params", "Query", "FormValue", "Get"),
            GoHTTPRequest.method("FormValue", "PostFormValue", "UserAgent"),
            GoHTTPRequest.attr("Body", "URL.Path", "URL.RawQuery"),
            GoChiRouter.method("URLParam"),
            GoGorillaMux.method("Vars"),
        ],
        to_sinks=[
            GoSQLDB.method(
                "Query", "Exec", "QueryRow",
                "QueryContext", "ExecContext", "QueryRowContext",
                "Prepare", "PrepareContext",
            ),
        ],
        sanitized_by=[
            GoStrconv.method("Atoi", "ParseInt", "ParseFloat", "ParseUint"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )


@go_rule(
    id="GO-SQLI-002",
    severity="CRITICAL",
    cwe="CWE-89",
    owasp="A03:2021",
    tags="go,security,sql-injection,pgx,postgresql,CWE-89,OWASP-A03",
    message=(
        "User-controlled input flows into a pgx query method (Exec, Query, QueryRow). "
        "This creates a SQL injection vulnerability in your PostgreSQL driver. "
        "Use pgx parameterized queries: conn.Exec(ctx, 'SELECT $1', userID)"
    ),
)
def go_pgx_sql_injection():
    """HTTP request input reaches pgx (PostgreSQL driver) query methods — SQL injection.

    pgx is the high-performance native PostgreSQL driver. Raw string queries are
    just as vulnerable as database/sql.

    Bad:  conn.Exec(ctx, "DELETE FROM sessions WHERE user = " + uid)
    Good: conn.Exec(ctx, "DELETE FROM sessions WHERE user = $1", uid)
    """
    return flows(
        from_sources=[
            GoGinContext.method(
                "Param", "Query", "PostForm", "GetRawData",
                "ShouldBindJSON", "BindJSON", "GetHeader"
            ),
            GoEchoContext.method("QueryParam", "FormValue", "Param"),
            GoFiberCtx.method("Params", "Query", "FormValue", "Get"),
            GoHTTPRequest.method("FormValue", "PostFormValue"),
            GoHTTPRequest.attr("Body", "URL.Path", "URL.RawQuery"),
        ],
        to_sinks=[
            GoPgxConn.method(
                "Exec", "Query", "QueryRow",
                "ExecEx", "QueryEx", "QueryRowEx",
                "SendBatch",
            ),
        ],
        sanitized_by=[
            GoStrconv.method("Atoi", "ParseInt", "ParseFloat", "ParseUint"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )


@go_rule(
    id="GO-SQLI-003",
    severity="CRITICAL",
    cwe="CWE-89",
    owasp="A03:2021",
    tags="go,security,sql-injection,sqlx,CWE-89,OWASP-A03",
    message=(
        "User-controlled input flows into a sqlx query method (Exec, Query, Get, Select). "
        "This creates a SQL injection vulnerability. "
        "Use named parameters: db.NamedExec('SELECT * WHERE id = :id', map[string]any{'id': id})"
    ),
)
def go_sqlx_sql_injection():
    """HTTP request input reaches sqlx (extended database/sql) query methods — SQL injection.

    sqlx extends database/sql with named params and struct scanning. The raw
    query methods are still vulnerable to SQL injection.

    Bad:  db.Get(&user, "SELECT * FROM users WHERE id = " + uid)
    Good: db.Get(&user, "SELECT * FROM users WHERE id = $1", uid)
    """
    return flows(
        from_sources=[
            GoGinContext.method(
                "Param", "Query", "PostForm", "GetRawData",
                "ShouldBindJSON", "BindJSON", "GetHeader"
            ),
            GoEchoContext.method("QueryParam", "FormValue", "Param"),
            GoHTTPRequest.method("FormValue", "PostFormValue"),
            GoHTTPRequest.attr("Body", "URL.Path", "URL.RawQuery"),
        ],
        to_sinks=[
            GoSqlxDB.method(
                "Exec", "Query", "QueryRow", "Get", "Select",
                "NamedExec", "NamedQuery",
                "ExecContext", "QueryContext", "QueryRowContext",
                "GetContext", "SelectContext",
                "MustExec", "Queryx", "QueryRowx",
            ),
        ],
        sanitized_by=[
            GoStrconv.method("Atoi", "ParseInt", "ParseFloat", "ParseUint"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
