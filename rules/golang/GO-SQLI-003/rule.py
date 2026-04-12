"""GO-SQLI-003: SQL injection via sqlx (extended database/sql)."""

from codepathfinder.go_rule import (
    GoGinContext,
    GoHTTPRequest,
    GoSqlxDB,
    GoEchoContext,
    GoStrconv,
)
from codepathfinder import flows
from codepathfinder.presets import PropagationPresets
from rules.go_decorators import go_rule


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
    """HTTP request input reaches sqlx (extended database/sql) query methods."""
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
