"""GO-SSRF-002: SSRF via HTTP request input reaching outbound net/http client calls."""

from codepathfinder.go_rule import (
    GoGinContext,
    GoHTTPRequest,
    GoHTTPClient,
    GoEchoContext,
    GoFiberCtx,
)
from codepathfinder import flows
from codepathfinder.presets import PropagationPresets
from rules.go_decorators import go_rule


@go_rule(
    id="GO-SSRF-002",
    severity="HIGH",
    cwe="CWE-918",
    owasp="A10:2021",
    tags="go,security,ssrf,net-http,CWE-918,OWASP-A10",
    message=(
        "User-controlled input flows into net/http client methods (http.Get, http.Post, "
        "http.NewRequest, client.Do). This creates a Server-Side Request Forgery (SSRF) "
        "vulnerability. Attackers can route requests to internal services, cloud metadata "
        "endpoints, or arbitrary hosts. Validate the URL host against an allowlist."
    ),
)
def go_ssrf_http_client():
    """HTTP request input reaches outbound net/http client — SSRF."""
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
            GoHTTPClient.method("Get", "Post", "Do", "Head", "PostForm"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
