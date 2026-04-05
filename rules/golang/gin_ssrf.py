"""
GO-SSRF-001: Server-Side Request Forgery via User-Controlled URLs

Security Impact: HIGH
CWE: CWE-918 (Server-Side Request Forgery)
OWASP: A10:2021 (Server-Side Request Forgery)

DESCRIPTION:
When user-controlled input from Gin or net/http request parameters flows into
HTTP client methods (resty, net/http), it creates a Server-Side Request Forgery
(SSRF) vulnerability. Attackers can make the server issue requests to internal
services, cloud metadata endpoints, or other unintended destinations.

VULNERABLE EXAMPLE:
```go
func proxyHandler(c *gin.Context) {
    target := c.Query("target")
    // CRITICAL: SSRF — user controls the URL
    resp, _ := resty.New().R().Get(target)
    c.String(200, resp.String())
}
```

SECURE EXAMPLE:
```go
func proxyHandler(c *gin.Context) {
    target := c.Query("target")
    // SECURE: Validate against an allowlist of known safe URLs
    if !isAllowedURL(target) {
        c.AbortWithStatus(400)
        return
    }
    resp, _ := resty.New().R().Get(target)
    c.String(200, resp.String())
}
```

BEST PRACTICES:
1. Validate URLs against an explicit allowlist before making outbound requests
2. Block requests to private IP ranges (10.x.x.x, 172.16.x.x, 192.168.x.x, 127.x.x.x)
3. Block requests to cloud metadata endpoints (169.254.169.254)
4. Use a dedicated SSRF-protection library
5. Apply network-level egress filtering

REFERENCES:
- CWE-918: https://cwe.mitre.org/data/definitions/918.html
- OWASP SSRF: https://owasp.org/Top10/A10_2021-Server-Side_Request_Forgery_%28SSRF%29/
- OWASP SSRF Prevention: https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html
"""

from codepathfinder.go_rule import (
    GoGinContext,
    GoRestyClient,
    GoHTTPRequest,
)
from codepathfinder import flows
from codepathfinder.presets import PropagationPresets
from rules.go_decorators import go_rule


@go_rule(id="GO-SSRF-001", severity="HIGH", cwe="CWE-918", owasp="A10:2021")
def detect_gin_ssrf():
    """Detect SSRF via user-controlled URLs in HTTP client calls.

    When user input from Gin request parameters flows into HTTP client
    methods (resty, net/http), it creates a Server-Side Request Forgery
    vulnerability.

    Bad: url := c.Query("target"); resty.New().R().Get(url)
    """
    return flows(
        from_sources=[
            GoGinContext.method("Query", "Param", "PostForm"),
            GoHTTPRequest.method("FormValue"),
        ],
        to_sinks=[
            GoRestyClient.method("Get", "Post", "Put", "Delete", "SetBaseURL"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
