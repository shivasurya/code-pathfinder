"""
Python Django Command Injection Rules

Rules:
- PYTHON-DJANGO-SEC-010: Command Injection via os.system() (CWE-78)
- PYTHON-DJANGO-SEC-011: Command Injection via subprocess (CWE-78)

Security Impact: CRITICAL
CWE: CWE-78 (Improper Neutralization of Special Elements used in an OS Command)
OWASP: A03:2021 - Injection

DESCRIPTION:
These rules detect OS command injection vulnerabilities in Django applications where
untrusted user input from HTTP requests flows into system command execution functions
such as os.system(), os.popen(), or subprocess module calls. When user-controlled data
is incorporated into shell commands without proper sanitization, attackers can inject
arbitrary OS commands that execute with the privileges of the web application process.

SECURITY IMPLICATIONS:

**1. Remote Code Execution (RCE)**:
An attacker can execute arbitrary system commands on the server by injecting shell
metacharacters (;, |, &&, ``, $()) into user input that reaches command execution functions.

**2. System Compromise**:
Successful command injection gives attackers full control over the server, allowing them
to install backdoors, pivot to internal networks, steal credentials, or modify system
configurations.

**3. Data Exfiltration**:
Attackers can use commands like curl, wget, or netcat to exfiltrate sensitive files,
database dumps, environment variables, and API keys to external servers.

**4. Denial of Service**:
Injected commands can crash services, consume resources (fork bombs), or delete critical
files, causing extended downtime.

VULNERABLE EXAMPLE:
```python
import os
import subprocess


# SEC-010: os.system with request data
def vulnerable_os_system(request):
    filename = request.GET.get('file')
    os.system(f"cat {filename}")


# SEC-011: subprocess with request data
def vulnerable_subprocess(request):
    cmd = request.POST.get('command')
    subprocess.call(cmd, shell=True)


def vulnerable_subprocess_popen(request):
    host = request.GET.get('host')
    proc = subprocess.Popen(f"ping {host}", shell=True)
    return proc.communicate()
```

SECURE EXAMPLE:
```python
from django.http import JsonResponse
import subprocess
import shlex
import re

def ping_host(request):
    # SECURE: Validate input and use subprocess with list args
    host = request.GET.get('host', '')
    # Whitelist validation: only allow valid hostnames/IPs
    if not re.match(r'^[a-zA-Z0-9._-]+$', host):
        return JsonResponse({'error': 'Invalid host'}, status=400)
    result = subprocess.run(['ping', '-c', '3', host], capture_output=True, text=True)
    return JsonResponse({'result': result.stdout})

def run_tool(request):
    # SECURE: Use list arguments, never shell=True with user input
    filename = request.POST.get('filename', '')
    if not re.match(r'^[a-zA-Z0-9._-]+$', filename):
        return JsonResponse({'error': 'Invalid filename'}, status=400)
    result = subprocess.run(['file', filename], capture_output=True, text=True)
    return JsonResponse({'result': result.stdout})
```

DETECTION AND PREVENTION:

**Key Mitigation Strategies**:
- Never pass user input to os.system(), os.popen(), or shell=True subprocess calls
- Use subprocess.run() with list arguments instead of string commands
- Apply shlex.quote() when shell execution is absolutely necessary
- Validate and whitelist user input with strict regex patterns
- Use allowlists for permitted commands and arguments
- Run application processes with minimal OS privileges

**Pre-deployment checks**:
```bash
pathfinder scan --project . --ruleset cpf/python/django/command-injection
```

COMPLIANCE:
- CWE-78: Improper Neutralization of Special Elements used in an OS Command
- OWASP A03:2021 - Injection
- SANS Top 25: CWE-78 ranked #5
- NIST SP 800-53: SI-10 (Information Input Validation)

REFERENCES:
- CWE-78: https://cwe.mitre.org/data/definitions/78.html
- OWASP Command Injection: https://owasp.org/www-community/attacks/Command_Injection
- Python subprocess documentation: https://docs.python.org/3/library/subprocess.html
- Django Security: https://docs.djangoproject.com/en/stable/topics/security/
"""

from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets


class OSModule(QueryType):
    fqns = ["os"]


class SubprocessModule(QueryType):
    fqns = ["subprocess"]


_DJANGO_SOURCES = [
    calls("request.GET.get"),
    calls("request.POST.get"),
    calls("request.GET"),
    calls("request.POST"),
    calls("request.COOKIES.get"),
    calls("request.FILES.get"),
    calls("*.GET.get"),
    calls("*.POST.get"),
]


@python_rule(
    id="PYTHON-DJANGO-SEC-010",
    name="Django Command Injection via os.system()",
    severity="CRITICAL",
    category="django",
    cwe="CWE-78",
    tags="python,django,command-injection,os-system,owasp-a03,cwe-78",
    message="User input flows to os.system(). Use subprocess with list args and shlex.quote().",
    owasp="A03:2021",
)
def detect_django_os_system_injection():
    """Detects Django request data flowing to os.system()/os.popen()."""
    return flows(
        from_sources=_DJANGO_SOURCES,
        to_sinks=[
            OSModule.method("system", "popen", "popen2", "popen3", "popen4").tracks(0),
        ],
        sanitized_by=[
            calls("shlex.quote"),
            calls("shlex.split"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )


@python_rule(
    id="PYTHON-DJANGO-SEC-011",
    name="Django Command Injection via subprocess",
    severity="CRITICAL",
    category="django",
    cwe="CWE-78",
    tags="python,django,command-injection,subprocess,owasp-a03,cwe-78",
    message="User input flows to subprocess call. Use shlex.quote() or avoid shell=True.",
    owasp="A03:2021",
)
def detect_django_subprocess_injection():
    """Detects Django request data flowing to subprocess functions."""
    return flows(
        from_sources=_DJANGO_SOURCES,
        to_sinks=[
            SubprocessModule.method("call", "check_call", "check_output",
                                    "run", "Popen", "getoutput", "getstatusoutput").tracks(0),
        ],
        sanitized_by=[
            calls("shlex.quote"),
            calls("shlex.split"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
