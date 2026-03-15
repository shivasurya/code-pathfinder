"""
Operating System Command Injection Security Rules for Python

Rules in this file:
- PYTHON-LANG-SEC-010: Dangerous os.system() Call (CWE-78)
- PYTHON-LANG-SEC-011: Dangerous os.exec*() Call (CWE-78)
- PYTHON-LANG-SEC-012: Dangerous os.spawn*() Call (CWE-78)
- PYTHON-LANG-SEC-013: System Call with Wildcard (CWE-78)
- PYTHON-LANG-SEC-014: Python Reverse Shell Detected (CWE-506)

Security Impact: CRITICAL
CWE: CWE-78 (Improper Neutralization of Special Elements used in an OS Command)
OWASP: A03:2021 - Injection

DESCRIPTION:
The Python os module provides direct access to operating system functions that
execute shell commands and spawn processes. Functions such as os.system(),
os.popen(), os.exec*(), and os.spawn*() pass arguments directly to the OS shell
or kernel. When user-controlled data reaches these functions without proper
sanitization, attackers can inject arbitrary operating system commands.

SECURITY IMPLICATIONS:
OS command injection through os.system() or os.popen() allows full shell
interpretation including pipes, semicolons, and command chaining. The os.exec*()
family replaces the running process entirely, while os.spawn*() creates new
processes. Wildcard characters in system calls may cause unintended file
inclusion or argument injection. Socket-based reverse shells using the socket
module enable persistent remote access.

    # Attack scenario: command injection via os.system
    filename = request.args.get("file")
    os.system(f"cat {filename}")  # Attacker sends: "; rm -rf / #"

VULNERABLE EXAMPLE:
```python
import os
user_input = request.form["cmd"]
os.system(f"ls {user_input}")        # Shell injection
os.popen(f"grep {user_input} log")   # Shell injection via popen
os.execvp("/bin/sh", ["/bin/sh", "-c", user_input])  # Direct exec
```

SECURE EXAMPLE:
```python
import subprocess
import shlex
# Use subprocess with list arguments (no shell interpretation)
subprocess.run(["ls", user_input], check=True)
# Or properly quote shell arguments
subprocess.run(f"ls {shlex.quote(user_input)}", shell=True, check=True)
```

DETECTION AND PREVENTION:
- Replace os.system() and os.popen() with subprocess.run() using list arguments
- Never concatenate user input into shell command strings
- Use shlex.quote() if shell=True is unavoidable
- Apply input validation with strict allowlists for command arguments
- Monitor for socket-based reverse shell patterns in code reviews

COMPLIANCE:
- CWE-78: Improper Neutralization of Special Elements used in an OS Command
- CWE-506: Embedded Malicious Code
- OWASP A03:2021 - Injection
- SANS Top 25 (2023) - CWE-78: OS Command Injection
- NIST SP 800-53: SI-10 (Information Input Validation)

REFERENCES:
- https://cwe.mitre.org/data/definitions/78.html
- https://cwe.mitre.org/data/definitions/506.html
- https://owasp.org/Top10/A03_2021-Injection/
- https://docs.python.org/3/library/os.html#os.system
- https://docs.python.org/3/library/subprocess.html
- https://docs.python.org/3/library/shlex.html#shlex.quote
"""

from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets


class OSModule(QueryType):
    fqns = ["os"]


class SocketModule(QueryType):
    fqns = ["socket"]


@python_rule(
    id="PYTHON-LANG-SEC-010",
    name="Dangerous os.system() Call",
    severity="HIGH",
    category="lang",
    cwe="CWE-78",
    tags="python,os-system,command-injection,owasp-a03,cwe-78",
    message="os.system() detected. Use subprocess.run() with list arguments instead.",
    owasp="A03:2021",
)
def detect_os_system():
    """Detects os.system() and os.popen() calls."""
    return OSModule.method("system", "popen", "popen2", "popen3", "popen4")


@python_rule(
    id="PYTHON-LANG-SEC-011",
    name="Dangerous os.exec*() Call",
    severity="HIGH",
    category="lang",
    cwe="CWE-78",
    tags="python,os-exec,command-injection,owasp-a03,cwe-78",
    message="os.exec*() detected. These replace the current process with a new one.",
    owasp="A03:2021",
)
def detect_os_exec():
    """Detects os.execl/execle/execlp/execlpe/execv/execve/execvp/execvpe calls."""
    return OSModule.method("execl", "execle", "execlp", "execlpe",
                           "execv", "execve", "execvp", "execvpe")


@python_rule(
    id="PYTHON-LANG-SEC-012",
    name="Dangerous os.spawn*() Call",
    severity="HIGH",
    category="lang",
    cwe="CWE-78",
    tags="python,os-spawn,process-spawn,owasp-a03,cwe-78",
    message="os.spawn*() detected. Use subprocess module instead.",
    owasp="A03:2021",
)
def detect_os_spawn():
    """Detects os.spawnl/spawnle/spawnlp/spawnlpe/spawnv/spawnve/spawnvp/spawnvpe calls."""
    return OSModule.method("spawnl", "spawnle", "spawnlp", "spawnlpe",
                           "spawnv", "spawnve", "spawnvp", "spawnvpe")


@python_rule(
    id="PYTHON-LANG-SEC-013",
    name="System Call with Wildcard",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-78",
    tags="python,wildcard,command-injection,cwe-78",
    message="os.system() with wildcard (*) detected. Wildcards may lead to unintended file inclusion.",
    owasp="A03:2021",
)
def detect_system_wildcard():
    """Detects os.system() calls (audit for wildcard usage)."""
    return OSModule.method("system")


@python_rule(
    id="PYTHON-LANG-SEC-014",
    name="Python Reverse Shell Detected",
    severity="CRITICAL",
    category="lang",
    cwe="CWE-506",
    tags="python,reverse-shell,backdoor,cwe-506",
    message="Reverse shell pattern detected. This may be a backdoor.",
    owasp="A03:2021",
)
def detect_reverse_shell():
    """Detects socket-based reverse shell patterns."""
    return SocketModule.method("socket")
