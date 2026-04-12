"""
GO-XSS Rules: Cross-Site Scripting via unsafe template types and ResponseWriter writes.

GO-XSS-001: Unsafe html/template type conversions (template.HTML, template.CSS, etc.)
GO-XSS-002: User input written via fmt.Fprintf/Fprint/Fprintln to http.ResponseWriter
GO-XSS-003: User input written via io.WriteString to http.ResponseWriter

Security Impact: HIGH
CWE: CWE-79 (Improper Neutralization of Input During Web Page Generation)
OWASP: A03:2021 — Injection

DESCRIPTION:
Go's html/template package auto-escapes data by default. Developers bypass this
safety mechanism by explicitly converting user input to the trusted types
template.HTML, template.CSS, template.JS, template.URL, template.HTMLAttr, etc.
Any user-controlled data reaching these conversions causes XSS.

Similarly, writing unescaped user data directly to an http.ResponseWriter via
fmt.Fprintf or io.WriteString bypasses the template engine entirely.

VULNERABLE EXAMPLES:
    // XSS via type bypass
    name := r.FormValue("name")
    safe := template.HTML(name)   // CRITICAL: user data bypasses auto-escaping
    tmpl.Execute(w, safe)

    // XSS via direct write
    query := r.FormValue("q")
    fmt.Fprintf(w, "<p>Results for: "+query+"</p>")  // CRITICAL: unescaped output

SECURE EXAMPLES:
    // Use template engine properly — let it auto-escape
    name := r.FormValue("name")
    tmpl.Execute(w, name)  // SAFE: html/template auto-escapes

    // Use structured response (JSON API)
    json.NewEncoder(w).Encode(map[string]string{"q": query})

REFERENCES:
- CWE-79: https://cwe.mitre.org/data/definitions/79.html
- html/template safety: https://pkg.go.dev/html/template#hdr-Contexts
- OWASP XSS Prevention: https://cheatsheetseries.owasp.org/cheatsheets/Cross_Site_Scripting_Prevention_Cheat_Sheet.html
"""

from codepathfinder.go_rule import (
    GoHTTPRequest,
    GoGinContext,
    GoEchoContext,
    GoFiberCtx,
    GoFmt,
    GoIO,
    QueryType,
)
from codepathfinder import flows
from codepathfinder.presets import PropagationPresets
from codepathfinder.go_decorators import go_rule


class GoHTMLTemplateTypes(QueryType):
    """html/template unsafe type conversion functions that bypass auto-escaping."""

    fqns = ["html/template"]
    patterns = ["template.*"]
    match_subclasses = False


class GoHTTPResponseWriterSink(QueryType):
    """http.ResponseWriter — direct write sink for XSS."""

    fqns = ["net/http.ResponseWriter"]
    patterns = ["*.ResponseWriter"]
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
    """Detect user input flowing into html/template unsafe type conversions.

    Go's html/template auto-escapes all template data. Using template.HTML(),
    template.CSS(), template.JS(), template.URL() etc. overrides this safety
    and lets raw bytes through to the browser.

    Bad:  template.HTML(r.FormValue("name"))
    Good: tmpl.Execute(w, r.FormValue("name"))  // auto-escaped
    """
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
    """Detect user input flowing into fmt formatting functions that write to ResponseWriter.

    fmt.Fprintf(w, "...", userInput) writes unescaped content to the HTTP response.
    Any user-controlled data in the format args causes XSS.

    Bad:  fmt.Fprintf(w, "<p>"+r.FormValue("q")+"</p>")
    Good: tmpl.Execute(w, r.FormValue("q"))
    """
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
    """Detect user input flowing into io.WriteString targeting ResponseWriter.

    io.WriteString(w, userInput) bypasses all HTML escaping.

    Bad:  io.WriteString(w, r.FormValue("msg"))
    Good: tmpl.Execute(w, r.FormValue("msg"))
    """
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
