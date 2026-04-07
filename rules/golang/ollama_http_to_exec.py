"""
OLLAMA-SEC-001: HTTP / Gin Request Input Reaching OS Command Execution

Sources: gin.Context params/body, net/http.Request body
Sinks:   exec.Command, exec.CommandContext

Variants targeted:
- Model name from /api/pull, /api/chat JSON body → runner subprocess args
  (llm/server.go, x/imagegen/server.go, x/mlxrunner/client.go)
- Tool call command string → bash -c (x/tools/bash.go)
- Path param → exec arg chain

L1: GoGinContext + GoHTTPRequest sources, GoOSExec sink — both sides QueryType.
"""

from codepathfinder.go_rule import GoGinContext, GoHTTPRequest, GoOSExec
from codepathfinder import flows
from codepathfinder.presets import PropagationPresets
from rules.go_decorators import go_rule


@go_rule(id="OLLAMA-SEC-001", severity="CRITICAL", cwe="CWE-78", owasp="A03:2021")
def ollama_http_to_exec():
    """HTTP/Gin request input reaches os/exec — command injection variants."""
    return flows(
        from_sources=[
            GoGinContext.method("Param", "Query", "PostForm", "GetRawData",
                                "ShouldBindJSON", "BindJSON", "GetHeader"),
            GoHTTPRequest.method("FormValue", "PostFormValue"),
            GoHTTPRequest.attr("Body", "URL.Path", "URL.RawQuery", "Header"),
        ],
        to_sinks=[
            GoOSExec.method("Command", "CommandContext"),
        ],
        sanitized_by=[],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
