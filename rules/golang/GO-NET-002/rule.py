"""GO-NET-002: gRPC client using insecure connection (grpc.WithInsecure/WithNoTLS)."""

from codepathfinder.go_rule import QueryType
from codepathfinder import flows
from codepathfinder.go_decorators import go_rule


class GoGRPC(QueryType):
    fqns = ["google.golang.org/grpc"]
    patterns = ["grpc.*"]
    match_subclasses = False


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
    """Detect gRPC client connecting without TLS (WithInsecure / WithNoTLS)."""
    return GoGRPC.method("WithInsecure", "WithNoTLS")
