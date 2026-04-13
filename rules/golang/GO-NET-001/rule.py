"""GO-NET-001: HTTP server started without TLS (http.ListenAndServe)."""

from codepathfinder.go_rule import QueryType
from codepathfinder import flows
from codepathfinder.go_decorators import go_rule


class GoHTTPServer(QueryType):
    fqns = ["net/http"]
    patterns = ["http.*"]
    match_subclasses = False


@go_rule(
    id="GO-NET-001",
    severity="HIGH",
    cwe="CWE-319",
    owasp="A02:2021",
    tags="go,security,tls,http,cleartext,CWE-319,OWASP-A02",
    message=(
        "Detected http.ListenAndServe() starting an HTTP server without TLS. "
        "All data transmitted over HTTP is unencrypted and can be intercepted "
        "by network observers (man-in-the-middle attacks). "
        "Use http.ListenAndServeTLS(addr, certFile, keyFile, handler) instead."
    ),
)
def detect_http_without_tls():
    """Detect HTTP server started without TLS (http.ListenAndServe)."""
    return GoHTTPServer.method("ListenAndServe")
