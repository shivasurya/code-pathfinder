"""
Code Injection and Dynamic Execution Security Rules for Python

Rules in this file:
- PYTHON-LANG-SEC-001: Dangerous eval() Usage (CWE-95)
- PYTHON-LANG-SEC-002: Dangerous exec() Usage (CWE-95)
- PYTHON-LANG-SEC-003: Dangerous code.InteractiveConsole Usage (CWE-95)
- PYTHON-LANG-SEC-004: Dangerous globals() Usage (CWE-96)
- PYTHON-LANG-SEC-005: Non-literal import Detected (CWE-95)
- PYTHON-LANG-SEC-006: Dangerous Annotations Usage (CWE-95)

Security Impact: HIGH
CWE: CWE-95 (Improper Neutralization of Directives in Dynamically Evaluated Code)
OWASP: A03:2021 - Injection

DESCRIPTION:
Python provides several built-in functions and modules that evaluate or execute
arbitrary code at runtime, including eval(), exec(), code.InteractiveConsole, and
dynamic imports via __import__() or importlib.import_module(). When these functions
receive untrusted input, attackers can achieve full remote code execution (RCE) on
the host system.

SECURITY IMPLICATIONS:
An attacker who controls the argument to eval() or exec() can execute arbitrary
Python statements, including importing os, subprocess, or socket modules to run
shell commands, exfiltrate data, or establish reverse shells. Dynamic imports allow
loading attacker-controlled modules. The globals() function, when passed to
templating or formatting calls, exposes the entire global namespace and may allow
attribute traversal attacks.

    # Attack scenario: eval with user input
    user_input = request.args.get("expr")
    result = eval(user_input)  # Attacker sends: __import__('os').system('rm -rf /')

VULNERABLE EXAMPLE:
```python
import code
import importlib
import typing

# SEC-001: eval
user_input = input("Enter expression: ")
result = eval(user_input)

# SEC-002: exec
code_str = "print('hello')"
exec(code_str)

# SEC-003: code.InteractiveConsole
console = code.InteractiveConsole()
code.interact()

# SEC-004: globals
def render(template, **kwargs):
    return template.format(**globals())

# SEC-005: non-literal import
module_name = "os"
mod = __import__(module_name)
mod2 = importlib.import_module(module_name)

# SEC-006: dangerous annotations
class Foo:
    x: "eval('malicious')" = 1

hints = typing.get_type_hints(Foo)
```

SECURE EXAMPLE:
```python
# Use ast.literal_eval for safe literal parsing
import ast
result = ast.literal_eval(user_expr)

# Whitelist allowed modules for dynamic imports
ALLOWED_MODULES = {"json", "math", "datetime"}
if module_name in ALLOWED_MODULES:
    mod = __import__(module_name)
```

DETECTION AND PREVENTION:
- Replace eval() with ast.literal_eval() for parsing Python literals safely
- Avoid exec() entirely; use structured alternatives (dictionaries, dispatch tables)
- Never pass globals() or locals() to string formatting or template engines
- Restrict dynamic imports to an explicit allowlist of trusted module names
- Use code review and static analysis to flag all occurrences of eval/exec

COMPLIANCE:
- CWE-95: Improper Neutralization of Directives in Dynamically Evaluated Code
- CWE-96: Improper Neutralization of Directives in Statically Saved Code
- OWASP A03:2021 - Injection
- SANS Top 25 (2023) - CWE-94: Improper Control of Generation of Code
- NIST SP 800-53: SI-10 (Information Input Validation)

REFERENCES:
- https://cwe.mitre.org/data/definitions/95.html
- https://cwe.mitre.org/data/definitions/96.html
- https://owasp.org/Top10/A03_2021-Injection/
- https://docs.python.org/3/library/functions.html#eval
- https://docs.python.org/3/library/ast.html#ast.literal_eval
- https://docs.python.org/3/library/code.html
"""

from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets


class Builtins(QueryType):
    fqns = ["builtins"]


class CodeModule(QueryType):
    fqns = ["code"]


@python_rule(
    id="PYTHON-LANG-SEC-001",
    name="Dangerous eval() Detected",
    severity="HIGH",
    category="lang",
    cwe="CWE-95",
    tags="python,eval,code-injection,owasp-a03,cwe-95",
    message="eval() detected. Avoid eval() on untrusted input. Use ast.literal_eval() for safe parsing.",
    owasp="A03:2021",
)
def detect_eval():
    """Detects eval() calls."""
    return Builtins.method("eval")


@python_rule(
    id="PYTHON-LANG-SEC-002",
    name="Dangerous exec() Detected",
    severity="HIGH",
    category="lang",
    cwe="CWE-95",
    tags="python,exec,code-injection,owasp-a03,cwe-95",
    message="exec() detected. Never execute dynamically constructed code.",
    owasp="A03:2021",
)
def detect_exec():
    """Detects exec() calls."""
    return Builtins.method("exec")


@python_rule(
    id="PYTHON-LANG-SEC-003",
    name="Dangerous code.InteractiveConsole Usage",
    severity="HIGH",
    category="lang",
    cwe="CWE-95",
    tags="python,code-run,interactive-console,cwe-95",
    message="code.InteractiveConsole/interact detected. This enables arbitrary code execution.",
    owasp="A03:2021",
)
def detect_code_run():
    """Detects code.InteractiveConsole and code.interact usage."""
    return CodeModule.method("InteractiveConsole", "interact", "compile_command")


@python_rule(
    id="PYTHON-LANG-SEC-004",
    name="Dangerous globals() Usage",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-96",
    tags="python,globals,code-injection,cwe-96",
    message="globals() passed to a function. This may allow arbitrary attribute access.",
    owasp="A03:2021",
)
def detect_globals():
    """Detects globals() being passed to functions."""
    return calls("globals")


@python_rule(
    id="PYTHON-LANG-SEC-005",
    name="Non-literal import Detected",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-95",
    tags="python,import,dynamic-import,cwe-95",
    message="__import__() or importlib.import_module() with non-literal argument detected.",
    owasp="A03:2021",
)
def detect_non_literal_import():
    """Detects dynamic imports via __import__ and importlib."""
    return calls("__import__", "importlib.import_module")


@python_rule(
    id="PYTHON-LANG-SEC-006",
    name="Dangerous Annotations Usage",
    severity="MEDIUM",
    category="lang",
    cwe="CWE-95",
    tags="python,annotations,pep-563,cwe-95",
    message="typing.get_type_hints() may execute string annotations. Be cautious with untrusted code.",
    owasp="A03:2021",
)
def detect_dangerous_annotations():
    """Detects typing.get_type_hints() which evaluates string annotations."""
    return calls("typing.get_type_hints", "get_type_hints")
