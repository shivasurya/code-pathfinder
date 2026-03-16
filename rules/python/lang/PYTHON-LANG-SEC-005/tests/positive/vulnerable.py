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
