"""GO-SSRF-001: Server-Side Request Forgery via user-controlled URLs in HTTP client calls."""

from codepathfinder.go_rule import (
    GoGinContext,
    GoHTTPRequest,
    GoHTTPClient,
    GoRestyClient,
    GoEchoContext,
    GoFiberCtx,
    QueryType,
)
from codepathfinder import flows
from codepathfinder.presets import PropagationPresets
from rules.go_decorators import go_rule


@go_rule(
    id="GO-SSRF-001",
    severity="HIGH",
    cwe="CWE-918",
    owasp="A10:2021",
    tags="go,security,ssrf,http-client,CWE-918,OWASP-A10",
    message=(
        "User-controlled input flows into an HTTP client method (http.Get, http.Post, "
        "resty.Get, etc.). This creates a Server-Side Request Forgery (SSRF) vulnerability — "
        "attackers can make the server issue requests to internal services, cloud metadata "
        "endpoints (169.254.169.254), or other unintended destinations. "
        "Validate URLs against an explicit allowlist before making outbound requests."
    ),
)
def detect_ssrf():
    """Detect SSRF via user-controlled URLs in HTTP client calls."""
    return flows(
        from_sources=[
            GoGinContext.method("Query", "Param", "PostForm", "GetHeader", "GetRawData"),
            GoEchoContext.method("QueryParam", "FormValue", "Param"),
            GoFiberCtx.method("Params", "Query", "FormValue", "Get"),
            GoHTTPRequest.method("FormValue", "PostFormValue"),
            GoHTTPRequest.attr("URL.RawQuery", "URL.Path", "Body"),
        ],
        to_sinks=[
            GoHTTPClient.method("Get", "Post", "Do", "Head"),
            GoRestyClient.method("Get", "Post", "Put", "Delete", "SetBaseURL"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
