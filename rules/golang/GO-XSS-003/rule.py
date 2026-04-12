"""GO-XSS-003: XSS via io.WriteString writing user input to ResponseWriter."""

from codepathfinder.go_rule import (
    GoHTTPRequest,
    GoGinContext,
    GoEchoContext,
    GoIO,
)
from codepathfinder import flows
from codepathfinder.presets import PropagationPresets
from rules.go_decorators import go_rule


@go_rule(
    id="GO-XSS-003",
    severity="HIGH",
    cwe="CWE-79",
    owasp="A03:2021",
    tags="go,security,xss,responsewriter,io,CWE-79,OWASP-A03",
    message=(
        "User-controlled input flows into io.WriteString writing to an http.ResponseWriter. "
        "This writes raw unescaped HTML to the browser and can result in Cross-Site Scripting (XSS). "
        "Use html/template.Execute() to render user-controlled data safely."
    ),
)
def detect_io_writestring_to_responsewriter():
    """Detect user input flowing into io.WriteString targeting ResponseWriter."""
    return flows(
        from_sources=[
            GoHTTPRequest.method(
                "FormValue", "PostFormValue", "UserAgent", "Referer", "RequestURI"
            ),
            GoHTTPRequest.attr("Body", "URL.Path", "URL.RawQuery", "Host"),
            GoGinContext.method("Param", "Query", "PostForm", "GetHeader"),
            GoEchoContext.method("QueryParam", "FormValue", "Param"),
        ],
        to_sinks=[
            GoIO.method("WriteString"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
