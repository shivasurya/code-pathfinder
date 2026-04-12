"""GO-XSS-002: XSS via fmt.Fprintf/Fprintln/Fprint writing user input to ResponseWriter."""

from codepathfinder.go_rule import (
    GoHTTPRequest,
    GoGinContext,
    GoEchoContext,
    GoFiberCtx,
    GoFmt,
)
from codepathfinder import flows
from codepathfinder.presets import PropagationPresets
from rules.go_decorators import go_rule


@go_rule(
    id="GO-XSS-002",
    severity="HIGH",
    cwe="CWE-79",
    owasp="A03:2021",
    tags="go,security,xss,responsewriter,fmt,CWE-79,OWASP-A03",
    message=(
        "User-controlled input flows into fmt.Fprintf/Fprintln/Fprint writing to "
        "an http.ResponseWriter. This bypasses html/template escaping and can result "
        "in Cross-Site Scripting (XSS). "
        "Use html/template to render user data, or call html.EscapeString() before writing."
    ),
)
def detect_fmt_write_to_responsewriter():
    """Detect user input flowing into fmt formatting functions writing to ResponseWriter."""
    return flows(
        from_sources=[
            GoHTTPRequest.method(
                "FormValue", "PostFormValue", "UserAgent", "Referer", "RequestURI"
            ),
            GoHTTPRequest.attr("Body", "URL.Path", "URL.RawQuery", "Host"),
            GoGinContext.method("Param", "Query", "PostForm", "GetHeader"),
            GoEchoContext.method("QueryParam", "FormValue", "Param"),
            GoFiberCtx.method("Params", "Query", "FormValue", "Get"),
        ],
        to_sinks=[
            GoFmt.method("Fprintf", "Fprintln", "Fprint"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
