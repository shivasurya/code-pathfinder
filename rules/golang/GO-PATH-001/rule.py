"""GO-PATH-001: Path traversal via HTTP request input reaching file system operations."""

from codepathfinder.go_rule import (
    GoGinContext,
    GoHTTPRequest,
    GoOS,
    GoFilepath,
)
from codepathfinder import flows
from codepathfinder.presets import PropagationPresets
from codepathfinder.go_decorators import go_rule


@go_rule(
    id="GO-PATH-001",
    severity="HIGH",
    cwe="CWE-22",
    owasp="A01:2021",
    tags="go,security,path-traversal,file-system,CWE-22,OWASP-A01",
    message=(
        "User-controlled input flows into file system operations (os.Open, os.Create, "
        "os.ReadFile, filepath.Join). This creates a path traversal vulnerability — "
        "attackers can read or write arbitrary files by injecting '../' sequences. "
        "Use filepath.Clean() and validate the result stays within the intended base directory."
    ),
)
def go_path_traversal():
    """HTTP request input reaches file system operations — path traversal."""
    return flows(
        from_sources=[
            GoGinContext.method(
                "Param", "Query", "PostForm", "GetRawData",
                "ShouldBindJSON", "BindJSON", "GetHeader"
            ),
            GoHTTPRequest.method("FormValue", "PostFormValue"),
            GoHTTPRequest.attr("URL.Path", "URL.RawQuery"),
        ],
        to_sinks=[
            GoOS.method(
                "Open", "OpenFile", "Create", "CreateTemp",
                "ReadFile", "WriteFile", "Remove", "RemoveAll",
                "Mkdir", "MkdirAll", "Stat", "Lstat",
            ),
            GoFilepath.method("Join", "Abs"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
