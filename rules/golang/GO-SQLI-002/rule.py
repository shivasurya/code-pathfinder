"""GO-SQLI-002: SQL injection via pgx (PostgreSQL native driver)."""

from codepathfinder.go_rule import (
    GoGinContext,
    GoHTTPRequest,
    GoPgxConn,
    GoEchoContext,
    GoFiberCtx,
    GoStrconv,
)
from codepathfinder import flows
from codepathfinder.presets import PropagationPresets
from codepathfinder.go_decorators import go_rule


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
    """HTTP request input reaches pgx (PostgreSQL driver) query methods — SQL injection."""
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
