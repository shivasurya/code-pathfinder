"""
OLLAMA-SEC-003: HTTP / Gin Request Input Reaching Outbound HTTP Calls (SSRF)

Sources: gin.Context params/body, net/http.Request
Sinks:   http.Client.Do, http.Get, http.Post, http.NewRequest, http.NewRequestWithContext

Variants targeted:
- Model RemoteHost field (from JSON body) → outbound HTTP call (SSRF via model config)
- Registry URL built from user-supplied model name → download request
- WWW-Authenticate realm forwarded to attacker host (CVE-2025-51471 variant)
- Cloud proxy path constructed from user model reference

L1: GoGinContext + GoHTTPRequest sources, GoHTTPClient sink — both sides QueryType.
"""

from codepathfinder.go_rule import GoGinContext, GoHTTPRequest, GoHTTPClient
from codepathfinder import flows
from codepathfinder.presets import PropagationPresets
from codepathfinder.go_decorators import go_rule


@go_rule(id="OLLAMA-SEC-003", severity="HIGH", cwe="CWE-918", owasp="A10:2021")
def ollama_http_to_outbound():
    """HTTP/Gin request input reaches outbound HTTP calls — SSRF variants."""
    return flows(
        from_sources=[
            GoGinContext.method("Param", "Query", "PostForm", "GetRawData",
                                "ShouldBindJSON", "BindJSON", "GetHeader"),
            GoHTTPRequest.method("FormValue", "PostFormValue"),
            GoHTTPRequest.attr("Body", "URL.Path", "URL.RawQuery", "Header"),
        ],
        to_sinks=[
            GoHTTPClient.method("Do", "Get", "Post", "Head",
                                "NewRequest", "NewRequestWithContext"),
        ],
        sanitized_by=[],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
