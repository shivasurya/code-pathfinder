"""
PYTHON-FLASK-SEC-004: Flask Code Injection via eval()
PYTHON-FLASK-SEC-005: Flask Code Injection via exec()

Security Impact: CRITICAL
CWE: CWE-95 (Improper Neutralization of Directives in Dynamically Evaluated Code)
OWASP: A03:2021 - Injection

DESCRIPTION:
These rules detect code injection vulnerabilities in Flask applications where user-controlled
input flows into Python's dynamic code evaluation functions. Two attack vectors are covered:

- **SEC-004**: Detects flows from Flask request data to eval(), which evaluates a string as
  a Python expression and returns its result. Attackers can execute arbitrary expressions
  including function calls, attribute access, and import statements.

- **SEC-005**: Detects flows from Flask request data to exec() and compile(), which execute
  arbitrary Python statements. Unlike eval(), exec() can execute multi-line code blocks,
  import modules, define classes, and perform any Python operation.

These are among the most dangerous injection vulnerabilities because they provide direct
arbitrary code execution within the Python interpreter, with the full privileges of the
application process.

SECURITY IMPLICATIONS:

**1. Arbitrary Code Execution**:
eval() and exec() execute Python code directly. An attacker supplying
`__import__('os').system('rm -rf /')` to an eval() sink gains full code execution
capability on the server.

**2. Sandbox Escape**:
Even restricted eval() environments can typically be escaped using Python's introspection
capabilities such as __builtins__, __subclasses__(), and __globals__ to access dangerous
functions.

**3. Data Theft and Manipulation**:
Attackers can read files, access databases, modify application state, steal secrets from
environment variables, and exfiltrate data through network connections.

**4. Persistent Backdoors**:
exec() allows writing files to disk, modifying import paths, and monkey-patching
application code at runtime to establish persistent access.

VULNERABLE EXAMPLE:
```python
from flask import Flask, request

app = Flask(__name__)

@app.route('/calc')
def calculator():
    expr = request.args.get('expr')
    result = eval(expr)
    return str(result)

@app.route('/run_code')
def run_code():
    code = request.form.get('code')
    exec(code)
    return "executed"
```

SECURE EXAMPLE:
```python
from flask import Flask, request
import ast
import json

app = Flask(__name__)

@app.route('/calculate')
def calculate():
    expression = request.args.get('expr')
    # SAFE: ast.literal_eval only evaluates literals (strings, numbers, tuples, etc.)
    try:
        result = ast.literal_eval(expression)
    except (ValueError, SyntaxError):
        return {'error': 'Invalid expression'}, 400
    return {'result': result}

# For math expressions, use a dedicated safe parser
# For structured data, use json.loads()
```

DETECTION AND PREVENTION:

**Pre-deployment checks**:
```bash
pathfinder scan --project . --ruleset cpf/python/PYTHON-FLASK-SEC-004,PYTHON-FLASK-SEC-005
```

**Code Review Checklist**:
- [ ] No eval() or exec() with any user-supplied input
- [ ] ast.literal_eval() used instead of eval() for parsing literal values
- [ ] json.loads() used for parsing structured data
- [ ] No compile() with user-controlled source strings
- [ ] Dedicated safe expression parsers for math/formula evaluation

COMPLIANCE:
- CWE-95: Improper Neutralization of Directives in Dynamically Evaluated Code
- OWASP Top 10 A03:2021 - Injection
- NIST SP 800-53 SI-10: Information Input Validation

REFERENCES:
- CWE-95: https://cwe.mitre.org/data/definitions/95.html
- Python eval() Security: https://nedbatchelder.com/blog/201206/eval_really_is_dangerous.html
- Python ast.literal_eval: https://docs.python.org/3/library/ast.html#ast.literal_eval
- OWASP Code Injection: https://owasp.org/www-community/attacks/Code_Injection

DETECTION SCOPE:
These rules perform inter-procedural taint analysis tracking data from Flask request sources
to eval(), exec(), and compile() sinks. SEC-004 recognizes ast.literal_eval() and json.loads()
as sanitizers. SEC-005 has no recognized sanitizers since there is no safe way to execute
arbitrary user-supplied code.
"""

from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets


class Builtins(QueryType):
    fqns = ["builtins"]


@python_rule(
    id="PYTHON-FLASK-SEC-004",
    name="Flask Code Injection via eval()",
    severity="CRITICAL",
    category="flask",
    cwe="CWE-95",
    tags="python,flask,code-injection,eval,rce,owasp-a03,cwe-95",
    message="User input flows to eval(). Use ast.literal_eval() for safe evaluation.",
    owasp="A03:2021",
)
def detect_flask_eval_injection():
    """Detects Flask request data flowing to eval()."""
    return flows(
        from_sources=[
            calls("request.args.get"),
            calls("request.form.get"),
            calls("request.values.get"),
            calls("request.get_json"),
        ],
        to_sinks=[
            Builtins.method("eval").tracks(0),
            calls("eval"),
        ],
        sanitized_by=[
            calls("ast.literal_eval"),
            calls("json.loads"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )


@python_rule(
    id="PYTHON-FLASK-SEC-005",
    name="Flask Code Injection via exec()",
    severity="CRITICAL",
    category="flask",
    cwe="CWE-95",
    tags="python,flask,code-injection,exec,rce,owasp-a03,cwe-95",
    message="User input flows to exec(). Never execute user-supplied code.",
    owasp="A03:2021",
)
def detect_flask_exec_injection():
    """Detects Flask request data flowing to exec()."""
    return flows(
        from_sources=[
            calls("request.args.get"),
            calls("request.form.get"),
            calls("request.values.get"),
            calls("request.get_json"),
        ],
        to_sinks=[
            Builtins.method("exec", "compile").tracks(0),
            calls("exec"),
            calls("compile"),
        ],
        sanitized_by=[],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
