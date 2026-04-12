"""GO-REDIRECT-001: Open redirect via user-controlled URL in http.Redirect."""

from codepathfinder.go_rule import (
    GoHTTPRequest,
    GoGinContext,
    GoEchoContext,
    GoFiberCtx,
    QueryType,
)
from codepathfinder import flows
from codepathfinder.presets import PropagationPresets
from rules.go_decorators import go_rule


class GoHTTPServer(QueryType):
    fqns = ["net/http"]
    patterns = ["http.*"]
    match_subclasses = False


class GoGinResponse(QueryType):
    fqns = ["github.com/gin-gonic/gin.Context"]
    patterns = ["*.Context"]
    match_subclasses = False


class GoEchoResponse(QueryType):
    fqns = ["github.com/labstack/echo/v4.Context"]
    patterns = ["*.Context"]
    match_subclasses = False


@go_rule(
    id="GO-REDIRECT-001",
    severity="HIGH",
    cwe="CWE-601",
    owasp="A01:2021",
    tags="go,security,open-redirect,CWE-601,OWASP-A01",
    message=(
        "User-controlled input flows into an HTTP redirect (http.Redirect, "
        "gin.Context.Redirect, echo.Context.Redirect). "
        "This creates an open redirect vulnerability — attackers can send users "
        "to malicious websites via phishing links. "
        "Validate the redirect URL against an allowlist, or restrict to relative paths only."
    ),
)
def detect_open_redirect():
    """Detect open redirect via user-controlled URL in HTTP redirect functions."""
    return flows(
        from_sources=[
            GoHTTPRequest.method(
                "FormValue", "PostFormValue", "UserAgent", "Referer", "RequestURI", "Cookie",
            ),
            GoHTTPRequest.attr("URL.Path", "URL.RawQuery", "Host", "URL"),
            GoGinContext.method("Query", "Param", "PostForm", "GetHeader"),
            GoEchoContext.method("QueryParam", "FormValue", "Param"),
            GoFiberCtx.method("Params", "Query", "FormValue", "Get"),
        ],
        to_sinks=[
            GoHTTPServer.method("Redirect"),
            GoGinResponse.method("Redirect"),
            GoEchoResponse.method("Redirect"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
