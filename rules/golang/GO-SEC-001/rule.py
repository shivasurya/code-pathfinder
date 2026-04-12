"""GO-SEC-001: SQL injection via database/sql (standard library)."""

from codepathfinder.go_rule import (
    GoGinContext,
    GoHTTPRequest,
    GoSQLDB,
    GoEchoContext,
    GoFiberCtx,
    GoStrconv,
    QueryType,
)
from codepathfinder import flows
from codepathfinder.presets import PropagationPresets
from rules.go_decorators import go_rule


class GoChiRouter(QueryType):
    fqns = ["github.com/go-chi/chi/v5"]
    patterns = ["chi.*"]
    match_subclasses = False


class GoGorillaMux(QueryType):
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
    """HTTP request input reaches database/sql query methods — SQL injection."""
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
