"""GO-GORM-SQLI-002: SQL injection via GORM query builder methods with user input."""

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
    id="GO-GORM-SQLI-002",
    severity="HIGH",
    cwe="CWE-89",
    owasp="A03:2021",
    tags="go,security,sql-injection,gorm,query-builder,CWE-89,OWASP-A03",
    message=(
        "User-controlled input flows into GORM query builder methods (Order, Group, Having, "
        "Where, Distinct, Select, Pluck) as raw SQL strings. "
        "Attackers can inject SQL via ORDER BY, GROUP BY, or WHERE clauses. "
        "Validate user input against an allowlist of permitted column names and sort directions."
    ),
)
def detect_gorm_query_builder_sqli():
    """Detect SQL injection via GORM query builder methods with user input."""
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
