"""
GO-SEC-003: Path Traversal via User Input

VULNERABILITY DESCRIPTION:
Path traversal (directory traversal) occurs when user input is used to construct
file paths without proper validation. Attackers can use ../ sequences to access
files outside the intended directory.

SEVERITY: HIGH
CWE: CWE-22 (Path Traversal)
OWASP: A01:2021 (Broken Access Control)

IMPACT:
- Unauthorized file access
- Reading sensitive configuration files
- Source code disclosure
- Credential theft
- System file manipulation

VULNERABLE PATTERNS:
- HTTP parameters used in os.Open()
- User input concatenated into file paths
- URL paths used in os.ReadFile()

SECURE PATTERNS:
- Use filepath.Clean() to normalize paths
- Validate paths against allowlist
- Use filepath.Base() to extract filename only
- Check filepath.HasPrefix() to ensure path is within allowed directory

EXAMPLE:
Vulnerable:
    filename := r.FormValue("file")
    os.Open("/uploads/" + filename)  // Can be ../../../etc/passwd

Secure:
    filename := filepath.Base(r.FormValue("file"))  // Strips ../
    os.Open(filepath.Join("/uploads", filename))

REFERENCES:
- https://owasp.org/www-community/attacks/Path_Traversal
- https://cwe.mitre.org/data/definitions/22.html
"""

from codepathfinder import rule, calls, flows

@rule(
    id="GO-SEC-003",
    severity="HIGH",
    cwe="CWE-22",
    owasp="A01:2021"
)
def go_path_traversal():
    """Detects user input used in file system paths without sanitization."""
    return flows(
        from_sources=[
            calls("net/http.Request.FormValue"),
            calls("net/http.Request.URL.Query"),
            calls("*gin.Context.Query"),
            calls("*gin.Context.Param"),
        ],
        to_sinks=[
            calls("os.Open"),
            calls("os.OpenFile"),
            calls("os.ReadFile"),
            calls("os.Create"),
            calls("os.MkdirAll"),
        ],
        scope="global"
    )
