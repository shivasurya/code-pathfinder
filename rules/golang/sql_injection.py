"""
GO-SEC-001: SQL Injection via HTTP Input

Sources: gin.Context params/body, net/http.Request
Sinks:   database/sql DB/Tx/Stmt — Query, Exec, QueryRow, *Context variants

L1: GoGinContext + GoHTTPRequest sources, GoSQLDB sink — both sides QueryType.
scope="global" — inter-procedural cross-file taint analysis.
"""

from codepathfinder.go_rule import GoGinContext, GoHTTPRequest, GoSQLDB
from codepathfinder import flows
from codepathfinder.presets import PropagationPresets
from rules.go_decorators import go_rule


@go_rule(id="GO-SEC-001", severity="CRITICAL", cwe="CWE-89", owasp="A03:2021")
def go_sql_injection():
    """HTTP/Gin request input reaches database/sql query methods — SQL injection.
    L1: QueryType both sides, scope=global."""
    return flows(
        from_sources=[
            GoGinContext.method("Param", "Query", "PostForm", "GetRawData",
                                "ShouldBindJSON", "BindJSON", "GetHeader"),
            GoHTTPRequest.method("FormValue", "PostFormValue"),
            GoHTTPRequest.attr("Body", "URL.Path", "URL.RawQuery"),
        ],
        to_sinks=[
            GoSQLDB.method("Query", "Exec", "QueryRow",
                           "QueryContext", "ExecContext", "QueryRowContext"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
