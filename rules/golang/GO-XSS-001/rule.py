"""GO-XSS-001: XSS via unsafe html/template type conversions."""

from codepathfinder.go_rule import (
    GoHTTPRequest,
    GoGinContext,
    GoEchoContext,
    GoFiberCtx,
    QueryType,
)
from codepathfinder import flows
from codepathfinder.presets import PropagationPresets
from codepathfinder.go_decorators import go_rule


class GoHTMLTemplateTypes(QueryType):
    """html/template unsafe type conversion functions."""
    fqns = ["html/template"]
    patterns = ["template.*"]
    match_subclasses = False


@go_rule(
    id="GO-XSS-001",
    severity="HIGH",
    cwe="CWE-79",
    owasp="A03:2021",
    tags="go,security,xss,template,CWE-79,OWASP-A03",
    message=(
        "User-controlled input flows into a template unsafe type conversion "
        "(template.HTML, template.CSS, template.JS, template.URL, etc.). "
        "These conversions bypass html/template's automatic escaping and can "
        "result in Cross-Site Scripting (XSS). "
        "Remove the explicit type conversion and let html/template escape the data automatically."
    ),
)
def detect_unsafe_template_type():
    """Detect user input flowing into html/template unsafe type conversions."""
    return flows(
        from_sources=[
            GoHTTPRequest.method(
                "FormValue", "PostFormValue", "UserAgent", "Referer", "RequestURI"
            ),
            GoHTTPRequest.attr("Body", "URL.Path", "URL.RawQuery", "Host"),
            GoGinContext.method("Param", "Query", "PostForm", "GetHeader", "GetRawData"),
            GoEchoContext.method("QueryParam", "FormValue", "Param", "PathParam"),
            GoFiberCtx.method("Params", "Query", "FormValue", "Get"),
        ],
        to_sinks=[
            GoHTMLTemplateTypes.method(
                "HTML", "CSS", "HTMLAttr", "JS", "JSStr", "Srcset", "URL"
            ),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
