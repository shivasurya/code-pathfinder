"""
GO-SEC-002: Command Injection via User Input

VULNERABILITY DESCRIPTION:
Command injection occurs when user-controlled input is passed to os/exec functions
without proper validation. Attackers can execute arbitrary system commands by
injecting shell metacharacters or command separators.

SEVERITY: CRITICAL
CWE: CWE-78 (OS Command Injection)
OWASP: A03:2021 (Injection)

IMPACT:
- Remote code execution
- System compromise
- Data exfiltration
- Privilege escalation
- Lateral movement in network

VULNERABLE PATTERNS:
- HTTP parameters used in os/exec.Command()
- User input concatenated into shell commands
- Form values passed to exec.CommandContext()

SECURE PATTERNS:
- Avoid exec.Command with user input
- Use allowlists for valid command arguments
- Sanitize input and reject shell metacharacters (semicolon, pipe, ampersand, dollar, backtick, etc)
- Use libraries that don't invoke shell (e.g., specific API calls instead of CLI tools)

REFERENCES:
- https://owasp.org/www-community/attacks/Command_Injection
- https://cwe.mitre.org/data/definitions/78.html
"""

from codepathfinder import rule, calls, flows

@rule(
    id="GO-SEC-002",
    severity="CRITICAL",
    cwe="CWE-78",
    owasp="A03:2021"
)
def go_command_injection():
    """Detects user input flowing to OS command execution functions."""
    return flows(
        from_sources=[
            calls("net/http.Request.FormValue"),
            calls("net/http.Request.URL.Query"),
            calls("*gin.Context.Query"),
            calls("*gin.Context.Param"),
        ],
        to_sinks=[
            calls("os/exec.Command"),
            calls("os/exec.CommandContext"),
        ],
        scope="global"
    )
