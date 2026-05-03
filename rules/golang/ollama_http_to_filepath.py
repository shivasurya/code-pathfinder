"""
OLLAMA-SEC-002: HTTP / Gin Request Input Reaching File System Operations

Sources: gin.Context params/body, net/http.Request
Sinks:   filepath.Join, os.Create, os.Open, os.OpenFile, os.WriteFile, os.ReadFile, os.MkdirAll

Variants targeted:
- Blob digest from URL path c.Param("digest") → manifest path → os.Create (CVE-2024-37032 regression)
- Model name from JSON body → manifest filepath construction
- req.Files paths from CreateHandler → os.Open
- Zip entry name → filepath.Join (ZipSlip: CVE-2024-7773, CVE-2024-45436)

L1: GoGinContext + GoHTTPRequest sources, GoFilepath + GoOS sinks — both sides QueryType.
"""

from codepathfinder.go_rule import GoGinContext, GoHTTPRequest, GoOS, GoFilepath
from codepathfinder import flows
from codepathfinder.presets import PropagationPresets
from codepathfinder.go_decorators import go_rule


@go_rule(id="OLLAMA-SEC-002", severity="HIGH", cwe="CWE-22", owasp="A01:2021")
def ollama_http_to_filepath():
    """HTTP/Gin request input reaches file system operations — path traversal variants."""
    return flows(
        from_sources=[
            # c.Param("digest") is the confirmed direct user-controlled source
            # that flows into manifest.BlobsPath() and file ops in ollama.
            # ShouldBindJSON(&req) fills structs — struct field propagation
            # is not yet covered by standard presets.
            GoGinContext.method("Param", "Query", "PostForm", "GetHeader"),
            GoHTTPRequest.method("FormValue", "PostFormValue"),
            GoHTTPRequest.attr("URL.Path", "URL.RawQuery"),
        ],
        to_sinks=[
            GoFilepath.method("Join"),
            GoOS.method("Create", "Open", "OpenFile", "WriteFile",
                        "ReadFile", "MkdirAll", "Remove", "Rename"),
        ],
        sanitized_by=[],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
