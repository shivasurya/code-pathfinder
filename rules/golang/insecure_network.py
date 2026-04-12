"""
GO-NET Rules: Insecure network configuration vulnerabilities.

GO-NET-001: HTTP server started without TLS (http.ListenAndServe)
GO-NET-002: gRPC client using insecure connection (grpc.WithInsecure / grpc.WithNoTLS)
GO-NET-003: Missing TLS MinVersion in tls.Config (allows TLS 1.0/1.1)

Security Impact: HIGH
CWE: CWE-319 (Cleartext Transmission of Sensitive Information)
     CWE-300 (Channel Accessible by Non-Endpoint)
     CWE-326 (Inadequate Encryption Strength)
OWASP: A02:2021 — Cryptographic Failures
       A07:2021 — Identification and Authentication Failures

DESCRIPTION:
HTTP without TLS transmits all data (credentials, session tokens, personal data)
in cleartext — trivially intercepted by network observers (MITM attacks).

gRPC connections with WithInsecure() transmit RPCs without encryption. An attacker
on the network can read and modify all RPC calls, including authentication metadata.

TLS configurations without MinVersion set may negotiate TLS 1.0 or 1.1 — both
deprecated protocols with known vulnerabilities (BEAST, POODLE).

VULNERABLE EXAMPLES:
    // HTTP without TLS
    http.ListenAndServe(":8080", mux)  // VULNERABLE: cleartext

    // gRPC insecure
    conn, _ := grpc.Dial(addr, grpc.WithInsecure())  // VULNERABLE

    // TLS MinVersion not set (allows TLS 1.0)
    tlsConfig := &tls.Config{}  // VULNERABLE: no MinVersion

SECURE EXAMPLES:
    // Use HTTPS
    http.ListenAndServeTLS(":443", certFile, keyFile, mux)

    // Use gRPC with TLS credentials
    creds := credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12})
    conn, _ := grpc.Dial(addr, grpc.WithTransportCredentials(creds))

    // Set minimum TLS version
    tlsConfig := &tls.Config{MinVersion: tls.VersionTLS13}

REFERENCES:
- CWE-319: https://cwe.mitre.org/data/definitions/319.html
- CWE-300: https://cwe.mitre.org/data/definitions/300.html
- OWASP TLS Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/TLS_Cipher_String_Cheat_Sheet.html
- gRPC TLS: https://blog.gopheracademy.com/advent-2019/go-grps-and-tls/
"""

from codepathfinder.go_rule import GoHTTPClient, QueryType
from codepathfinder import calls
from codepathfinder.go_decorators import go_rule


class GoHTTPServer(QueryType):
    """net/http — HTTP server functions."""

    fqns = ["net/http"]
    patterns = ["http.*"]
    match_subclasses = False


class GoGRPC(QueryType):
    """google.golang.org/grpc — gRPC client/server."""

    fqns = ["google.golang.org/grpc"]
    patterns = ["grpc.*"]
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
        "Use http.ListenAndServeTLS(addr, certFile, keyFile, handler) instead, "
        "or terminate TLS at a load balancer/reverse proxy in production."
    ),
)
def detect_http_without_tls():
    """Detect HTTP server started without TLS (http.ListenAndServe).

    http.ListenAndServe transmits all HTTP traffic in cleartext. Credentials,
    session tokens, and sensitive data are exposed to network observers.

    Bad:  http.ListenAndServe(":8080", mux)
    Good: http.ListenAndServeTLS(":443", certFile, keyFile, mux)
    """
    return GoHTTPServer.method("ListenAndServe")


@go_rule(
    id="GO-NET-002",
    severity="HIGH",
    cwe="CWE-300",
    owasp="A07:2021",
    tags="go,security,grpc,tls,insecure,CWE-300,OWASP-A07",
    message=(
        "Detected gRPC client using grpc.WithInsecure() or grpc.WithNoTLS(). "
        "This creates an unencrypted gRPC connection — all RPC calls, including "
        "authentication metadata (tokens, credentials), are transmitted in cleartext. "
        "Use grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})) instead."
    ),
)
def detect_grpc_insecure_connection():
    """Detect gRPC client connecting without TLS (WithInsecure / WithNoTLS).

    grpc.WithInsecure() disables transport security entirely. Deprecated in gRPC-Go
    v1.35+ in favor of grpc.WithNoTLS(), which is equally insecure. Both should be
    replaced with WithTransportCredentials().

    Bad:  grpc.Dial(addr, grpc.WithInsecure())
    Good: grpc.Dial(addr, grpc.WithTransportCredentials(credentials.NewTLS(tlsConf)))
    """
    return GoGRPC.method("WithInsecure", "WithNoTLS")
