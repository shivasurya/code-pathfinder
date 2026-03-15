"""
Python Django Code Injection Rules

Rules:
- PYTHON-DJANGO-SEC-020: Code Injection via eval() (CWE-95)
- PYTHON-DJANGO-SEC-021: Code Injection via exec() (CWE-95)
- PYTHON-DJANGO-SEC-022: Globals Misuse Code Execution (CWE-96)

Security Impact: CRITICAL
CWE: CWE-95 (Improper Neutralization of Directives in Dynamically Evaluated Code),
     CWE-96 (Improper Following of a URL Redirect)
OWASP: A03:2021 - Injection

DESCRIPTION:
These rules detect code injection vulnerabilities in Django applications where untrusted
user input from HTTP requests flows into dynamic code evaluation functions such as eval(),
exec(), or is used to index globals() for arbitrary function dispatch. Python's eval() and
exec() interpret strings as Python code, meaning any user-controlled input reaching these
functions allows an attacker to execute arbitrary Python code on the server with the full
privileges of the application process.

SECURITY IMPLICATIONS:

**1. Remote Code Execution (RCE)**:
An attacker can execute arbitrary Python code by injecting malicious expressions into eval()
or statements into exec(). For example, eval("__import__('os').system('id')") executes
shell commands directly from Python.

**2. Complete Server Compromise**:
Through code injection, attackers can import any module, access the file system, read
environment variables containing secrets, establish reverse shells, or modify application
behavior at runtime.

**3. Sandbox Escape**:
Even with attempted restrictions (e.g., restricted builtins), Python's introspection
capabilities (__subclasses__, __globals__, __builtins__) make sandbox escapes trivial.
There is no safe way to use eval() or exec() with untrusted input.

**4. Globals Dispatch Attack**:
Using globals()[user_input]() to dynamically call functions allows attackers to invoke
any function in the module's global scope, including internal or administrative functions
not intended to be user-accessible.

VULNERABLE EXAMPLE:
```python
from django.http import JsonResponse

def calculate(request):
    # VULNERABLE: User input passed directly to eval()
    expression = request.GET.get('expr')
    result = eval(expression)
    # Attack: ?expr=__import__('os').system('rm -rf /')
    return JsonResponse({'result': result})

def run_code(request):
    # VULNERABLE: User input passed to exec()
    code = request.POST.get('code')
    exec(code)
    # Attack: code=import socket; s=socket.socket()... (reverse shell)
    return JsonResponse({'status': 'done'})

def dispatch(request):
    # VULNERABLE: User input indexes globals()
    action = request.GET.get('action')
    result = globals()[action]()
    # Attack: ?action=admin_delete_all_users
    return JsonResponse({'result': result})
```

SECURE EXAMPLE:
```python
from django.http import JsonResponse
import ast
import operator

# Safe math operations whitelist
SAFE_OPS = {
    ast.Add: operator.add,
    ast.Sub: operator.sub,
    ast.Mult: operator.mul,
    ast.Div: operator.truediv,
}

def safe_eval_math(expr):
    \"\"\"Evaluate simple math expressions safely using AST parsing.\"\"\"
    tree = ast.parse(expr, mode='eval')
    # Only allow numbers and basic arithmetic
    for node in ast.walk(tree):
        if not isinstance(node, (ast.Expression, ast.BinOp, ast.Constant,
                                 ast.Add, ast.Sub, ast.Mult, ast.Div)):
            raise ValueError("Unsafe expression")
    return eval(compile(tree, '<string>', 'eval'))

def calculate(request):
    # SECURE: Use ast.literal_eval or custom safe parser
    expression = request.GET.get('expr', '')
    try:
        result = safe_eval_math(expression)
    except (ValueError, SyntaxError):
        return JsonResponse({'error': 'Invalid expression'}, status=400)
    return JsonResponse({'result': result})

def dispatch(request):
    # SECURE: Use explicit allowlist for dispatch
    ALLOWED_ACTIONS = {
        'list': list_items,
        'search': search_items,
        'count': count_items,
    }
    action = request.GET.get('action', '')
    handler = ALLOWED_ACTIONS.get(action)
    if handler is None:
        return JsonResponse({'error': 'Unknown action'}, status=400)
    return handler(request)
```

DETECTION AND PREVENTION:

**Key Mitigation Strategies**:
- Never use eval() or exec() with user-supplied input under any circumstances
- Use ast.literal_eval() for safely parsing Python literal expressions (strings, numbers, lists)
- Build explicit allowlists/dispatch tables instead of dynamic globals() lookup
- For math expressions, use a dedicated safe expression parser
- Apply strict input validation and type checking before any dynamic evaluation
- Consider using a sandboxed environment if dynamic evaluation is truly required

**Pre-deployment checks**:
```bash
pathfinder scan --project . --ruleset cpf/python/django/code-injection
```

COMPLIANCE:
- CWE-95: Improper Neutralization of Directives in Dynamically Evaluated Code
- CWE-96: Improper Following of a URL Redirect (Globals misuse for arbitrary dispatch)
- OWASP A03:2021 - Injection
- SANS Top 25: Code Injection
- NIST SP 800-53: SI-10 (Information Input Validation)

REFERENCES:
- CWE-95: https://cwe.mitre.org/data/definitions/95.html
- CWE-96: https://cwe.mitre.org/data/definitions/96.html
- OWASP Code Injection: https://owasp.org/www-community/attacks/Code_Injection
- Python ast.literal_eval: https://docs.python.org/3/library/ast.html#ast.literal_eval
- Django Security: https://docs.djangoproject.com/en/stable/topics/security/
"""

from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets


class Builtins(QueryType):
    fqns = ["builtins"]


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
    id="PYTHON-DJANGO-SEC-020",
    name="Django Code Injection via eval()",
    severity="CRITICAL",
    category="django",
    cwe="CWE-95",
    tags="python,django,code-injection,eval,owasp-a03,cwe-95",
    message="User input flows to eval(). Never use eval() with untrusted data.",
    owasp="A03:2021",
)
def detect_django_eval_injection():
    """Detects Django request data flowing to eval()."""
    return flows(
        from_sources=_DJANGO_SOURCES,
        to_sinks=[
            Builtins.method("eval").tracks(0),
            calls("eval"),
        ],
        sanitized_by=[
            calls("ast.literal_eval"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )


@python_rule(
    id="PYTHON-DJANGO-SEC-021",
    name="Django Code Injection via exec()",
    severity="CRITICAL",
    category="django",
    cwe="CWE-95",
    tags="python,django,code-injection,exec,owasp-a03,cwe-95",
    message="User input flows to exec(). Never use exec() with untrusted data.",
    owasp="A03:2021",
)
def detect_django_exec_injection():
    """Detects Django request data flowing to exec()."""
    return flows(
        from_sources=_DJANGO_SOURCES,
        to_sinks=[
            Builtins.method("exec").tracks(0),
            calls("exec"),
        ],
        sanitized_by=[],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )


@python_rule(
    id="PYTHON-DJANGO-SEC-022",
    name="Django Globals Misuse Code Execution",
    severity="HIGH",
    category="django",
    cwe="CWE-96",
    tags="python,django,code-injection,globals,owasp-a03,cwe-96",
    message="User input used to index globals(). This allows arbitrary code execution.",
    owasp="A03:2021",
)
def detect_django_globals_misuse():
    """Detects Django request data used with globals() for code execution."""
    return flows(
        from_sources=_DJANGO_SOURCES,
        to_sinks=[
            calls("globals"),
            calls("globals().get"),
        ],
        sanitized_by=[],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
