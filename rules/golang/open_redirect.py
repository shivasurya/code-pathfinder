"""
GO-REDIRECT-001: Open Redirect via user-controlled URL in http.Redirect.

Security Impact: HIGH
CWE: CWE-601 (URL Redirection to Untrusted Site / 'Open Redirect')
OWASP: A01:2021 — Broken Access Control
       A03:2021 — Injection

DESCRIPTION:
When user-controlled input flows into http.Redirect() without validation, an
attacker can redirect users to a malicious website. This enables phishing attacks —
the victim sees a trusted URL in their browser before being sent to an attacker's
site. Attackers use open redirects to harvest credentials or distribute malware.

Common attack vector: Send victim a link like:
    https://trusted.example.com/login?next=https://evil.com/phishing

The application redirects to evil.com thinking it's going back to "next" after login.

VULNERABLE EXAMPLES:
    func loginHandler(w http.ResponseWriter, r *http.Request) {
        next := r.FormValue("next")
        // CRITICAL: Open Redirect — attacker controls destination
        http.Redirect(w, r, next, http.StatusFound)
    }

    func redirectHandler(c *gin.Context) {
        target := c.Query("to")
        // CRITICAL: Open Redirect via Gin
        c.Redirect(http.StatusMovedPermanently, target)
    }

SECURE EXAMPLES:
    func loginHandler(w http.ResponseWriter, r *http.Request) {
        next := r.FormValue("next")
        // SECURE: Validate against allowlist of safe paths
        if !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") {
            next = "/dashboard"  // fallback to safe default
        }
        http.Redirect(w, r, next, http.StatusFound)
    }

REFERENCES:
- CWE-601: https://cwe.mitre.org/data/definitions/601.html
- OWASP Open Redirect: https://cheatsheetseries.owasp.org/cheatsheets/Unvalidated_Redirects_and_Forwards_Cheat_Sheet.html
"""

from codepathfinder.go_rule import (
    GoHTTPRequest,
    GoHTTPClient,
    GoGinContext,
    GoEchoContext,
    GoFiberCtx,
    QueryType,
)
from codepathfinder import flows
from codepathfinder.presets import PropagationPresets
from codepathfinder.go_decorators import go_rule


class GoHTTPServer(QueryType):
    """net/http — HTTP redirect and server functions."""

    fqns = ["net/http"]
    patterns = ["http.*"]
    match_subclasses = False


class GoGinResponse(QueryType):
    """gin.Context — response methods including Redirect."""

    fqns = ["github.com/gin-gonic/gin.Context"]
    patterns = ["*.Context"]
    match_subclasses = False


class GoEchoResponse(QueryType):
    """echo.Context — response methods including Redirect."""

    fqns = ["github.com/labstack/echo/v4.Context"]
    patterns = ["*.Context"]
    match_subclasses = False


@go_rule(
    id="GO-REDIRECT-001",
    severity="HIGH",
    cwe="CWE-601",
    owasp="A01:2021",
    tags="go,security,open-redirect,ssrf,CWE-601,OWASP-A01",
    message=(
        "User-controlled input flows into an HTTP redirect (http.Redirect, "
        "gin.Context.Redirect, echo.Context.Redirect). "
        "This creates an open redirect vulnerability — attackers can send users "
        "to malicious websites via phishing links. "
        "Validate the redirect URL against an allowlist, or restrict to relative "
        "paths only (must start with '/' and not '//')."
    ),
)
def detect_open_redirect():
    """Detect open redirect via user-controlled URL in HTTP redirect functions.

    User input flowing into http.Redirect() enables attackers to craft URLs
    that redirect victims to attacker-controlled sites.

    Bad:  http.Redirect(w, r, r.FormValue("next"), 302)
    Good: redirect to allowlisted relative path only
    """
    return flows(
        from_sources=[
            GoHTTPRequest.method(
                "FormValue",
                "PostFormValue",
                "UserAgent",
                "Referer",
                "RequestURI",
                "Cookie",
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
