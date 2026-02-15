"""
GO-SEC-003: Path Traversal via User Input

Security Impact: HIGH
CWE: CWE-22 (Path Traversal)
OWASP: A01:2021 (Broken Access Control)

DESCRIPTION:
This rule detects calls to file system functions (Open, ReadFile, Create, etc.)
that may access files. Path traversal (directory traversal) occurs when user input
is used to construct file paths without proper validation. Attackers can use ../
sequences to escape the intended directory and access sensitive files.

SECURITY IMPLICATIONS:
Path traversal vulnerabilities allow attackers to access files outside the intended
directory, which can lead to:

1. **Sensitive File Disclosure**: Access /etc/passwd, /etc/shadow, configuration files
2. **Source Code Disclosure**: Read application source code and discover vulnerabilities
3. **Credential Theft**: Access database credentials, API keys, private keys
4. **Configuration File Access**: Read and manipulate application settings
5. **Log Poisoning**: Read or write to log files to inject malicious content
6. **Arbitrary File Write**: Overwrite system files if write operations are allowed

VULNERABLE EXAMPLE:
```go
func downloadFile(w http.ResponseWriter, r *http.Request) {
    // CRITICAL: Path traversal vulnerability
    filename := r.FormValue("file")
    data, err := os.ReadFile("/uploads/" + filename)
    if err != nil {
        http.Error(w, "File not found", 404)
        return
    }
    w.Write(data)
}

// Attack: ?file=../../../etc/passwd
// Reads: /uploads/../../../etc/passwd -> /etc/passwd
```

SECURE EXAMPLE:
```go
func downloadFile(w http.ResponseWriter, r *http.Request) {
    filename := r.FormValue("file")

    // 1. Extract base filename only (removes directory components)
    safeFilename := filepath.Base(filename)

    // 2. Join paths safely
    fullPath := filepath.Join("/uploads", safeFilename)

    // 3. Clean the path to resolve any ../ sequences
    cleanPath := filepath.Clean(fullPath)

    // 4. Verify the path is still within allowed directory
    if !strings.HasPrefix(cleanPath, "/uploads/") {
        http.Error(w, "Invalid file path", 400)
        return
    }

    // 5. Read file
    data, err := os.ReadFile(cleanPath)
    if err != nil {
        http.Error(w, "File not found", 404)
        return
    }
    w.Write(data)
}
```

BEST PRACTICES:
1. **Use filepath.Base()**: Extract filename only, removing directory components
2. **Use filepath.Join()**: Safely join path components
3. **Use filepath.Clean()**: Normalize paths and resolve ../ sequences
4. **Validate with HasPrefix()**: Ensure final path is within allowed directory
5. **Allowlist filenames**: Only allow specific, known-safe filenames
6. **Avoid user input in paths**: Use database IDs instead of filenames

SECURE PATTERN COMPARISON:
```go
// VULNERABLE: Direct concatenation
path := "/uploads/" + userInput
os.ReadFile(path)

// BETTER: Use filepath.Base
path := filepath.Join("/uploads", filepath.Base(userInput))
os.ReadFile(path)

// BEST: Validate and check prefix
safeFile := filepath.Base(userInput)
fullPath := filepath.Clean(filepath.Join("/uploads", safeFile))
if !strings.HasPrefix(fullPath, "/uploads/") {
    return errors.New("invalid path")
}
os.ReadFile(fullPath)
```

DETECTION LIMITATIONS:
This rule uses pattern matching and flags ALL calls to file system functions.
It cannot determine if:
- The path uses proper validation
- The input is actually user-controlled

Manual review is required to verify if detected calls are vulnerable.

REMEDIATION:
1. Use filepath.Base() to extract filename from user input
2. Use filepath.Join() to safely construct paths
3. Use filepath.Clean() to normalize and remove ../ sequences
4. Validate final path with strings.HasPrefix() to ensure it's in allowed directory
5. Consider using file IDs instead of user-provided filenames
6. Implement allowlist of permitted files

REFERENCES:
- OWASP Path Traversal: https://owasp.org/www-community/attacks/Path_Traversal
- CWE-22: Path Traversal: https://cwe.mitre.org/data/definitions/22.html
- OWASP A01:2021 Broken Access Control: https://owasp.org/Top10/A01_2021-Broken_Access_Control/
- Go filepath package: https://pkg.go.dev/path/filepath
"""

from codepathfinder import rule, calls

@rule(
    id="GO-SEC-003",
    severity="HIGH",
    cwe="CWE-22",
    owasp="A01:2021"
)
def go_path_traversal():
    """Detects file system operations that may be vulnerable to path traversal.
    Flags calls to os.Open, os.ReadFile, and related functions."""
    return calls("*Open", "*OpenFile", "*ReadFile", "*Create", "*MkdirAll")
