"""
GO-SEC-002: Command Injection via User Input

Security Impact: CRITICAL
CWE: CWE-78 (OS Command Injection)
OWASP: A03:2021 (Injection)

DESCRIPTION:
This rule detects calls to os/exec functions (Command, CommandContext) that may
execute system commands. Command injection occurs when user-controlled input is
passed to these functions without proper validation. Attackers can execute
arbitrary system commands by injecting shell metacharacters or command separators.

SECURITY IMPLICATIONS:
Command injection allows attackers to execute arbitrary operating system commands
on the server, which can lead to:

1. **Remote Code Execution**: Execute any command the application user can run
2. **System Compromise**: Install backdoors, create new user accounts
3. **Data Exfiltration**: Access and transmit sensitive files to external servers
4. **Privilege Escalation**: Exploit SUID binaries or sudo misconfigurations
5. **Lateral Movement**: Use the compromised system to attack other systems
6. **Denial of Service**: Consume system resources or shut down services

VULNERABLE EXAMPLE:
```go
func handleConvert(w http.ResponseWriter, r *http.Request) {
    // CRITICAL: User input directly in command
    filename := r.FormValue("file")
    cmd := exec.Command("convert", filename, "output.png")
    output, _ := cmd.Output()
    w.Write(output)
}

// Attack: ?file=input.jpg;cat /etc/passwd
// Executes: convert input.jpg;cat /etc/passwd output.png
```

SECURE EXAMPLE:
```go
func handleConvert(w http.ResponseWriter, r *http.Request) {
    filename := r.FormValue("file")

    // 1. Validate against allowlist
    if !isValidFilename(filename) {
        http.Error(w, "Invalid filename", 400)
        return
    }

    // 2. Use filepath.Base to prevent directory traversal
    safeFilename := filepath.Base(filename)

    // 3. Pass as separate argument (not concatenated)
    cmd := exec.Command("convert", safeFilename, "output.png")
    output, err := cmd.Output()
    if err != nil {
        http.Error(w, "Conversion failed", 500)
        return
    }
    w.Write(output)
}

func isValidFilename(name string) bool {
    // Only allow alphanumeric and safe characters
    matched, _ := regexp.MatchString(`^[a-zA-Z0-9._-]+$`, name)
    return matched
}
```

BEST PRACTICES:
1. **Avoid user input in commands**: Use APIs or libraries instead of shell commands
2. **Allowlist validation**: Only allow specific, known-safe values
3. **Escape shell metacharacters**: Use shellquote libraries if commands are unavoidable
4. **Separate arguments**: Pass arguments individually, not as concatenated strings
5. **Least privilege**: Run commands with minimal required permissions
6. **Input validation**: Reject inputs containing: ; | & $ ` \ " ' < > ( ) { }

DETECTION LIMITATIONS:
This rule uses pattern matching and flags ALL calls to exec.Command and
exec.CommandContext. It cannot determine if the input is actually user-controlled.
Manual review is required to verify if detected calls are vulnerable.

REMEDIATION:
1. Remove user input from command execution
2. Use allowlist validation for any required user parameters
3. Use dedicated Go libraries instead of shelling out (e.g., image/jpeg instead of ImageMagick)
4. If commands are necessary, use shellquote.Quote() to escape arguments
5. Run commands in sandboxed environments with limited permissions

REFERENCES:
- OWASP Command Injection: https://owasp.org/www-community/attacks/Command_Injection
- CWE-78: OS Command Injection: https://cwe.mitre.org/data/definitions/78.html
- OWASP A03:2021 Injection: https://owasp.org/Top10/A03_2021-Injection/
"""

from codepathfinder import rule, calls

@rule(
    id="GO-SEC-002",
    severity="CRITICAL",
    cwe="CWE-78",
    owasp="A03:2021"
)
def go_command_injection():
    """Detects OS command execution that may be vulnerable to injection.
    Flags calls to exec.Command and exec.CommandContext."""
    return calls("*Command", "*CommandContext")
