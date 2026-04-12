"""GO-SEC-002: Command injection via HTTP request input reaching os/exec."""

from codepathfinder.go_rule import (
    GoGinContext,
    GoHTTPRequest,
    GoOSExec,
    GoEchoContext,
    GoFiberCtx,
)
from codepathfinder import flows
from codepathfinder.presets import PropagationPresets
from rules.go_decorators import go_rule


@go_rule(
    id="GO-SEC-002",
    severity="CRITICAL",
    cwe="CWE-78",
    owasp="A03:2021",
    tags="go,security,command-injection,os-exec,CWE-78,OWASP-A03",
    message=(
        "User-controlled input flows into os/exec command execution (exec.Command, "
        "exec.CommandContext). This creates a command injection vulnerability — attackers "
        "can execute arbitrary system commands on the server. "
        "Avoid shelling out with user input. If unavoidable, validate against a strict "
        "allowlist and pass arguments separately, never concatenated."
    ),
)
def go_command_injection():
    """HTTP request input reaches os/exec — command injection."""
    return flows(
        from_sources=[
            GoGinContext.method(
                "Param", "Query", "PostForm", "GetRawData",
                "ShouldBindJSON", "BindJSON", "GetHeader"
            ),
            GoEchoContext.method("QueryParam", "FormValue", "Param", "PathParam"),
            GoFiberCtx.method("Params", "Query", "FormValue", "Get"),
            GoHTTPRequest.method("FormValue", "PostFormValue", "UserAgent"),
            GoHTTPRequest.attr("Body", "URL.Path", "URL.RawQuery", "Header"),
        ],
        to_sinks=[
            GoOSExec.method("Command", "CommandContext"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
