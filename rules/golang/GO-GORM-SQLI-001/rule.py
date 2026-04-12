"""GO-GORM-SQLI-001: SQL injection via GORM Raw/Exec with user-controlled input."""

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
from rules.go_decorators import go_rule


class GoChiRouter(QueryType):
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
    """Detect SQL injection via GORM Raw/Exec with user-controlled input."""
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
