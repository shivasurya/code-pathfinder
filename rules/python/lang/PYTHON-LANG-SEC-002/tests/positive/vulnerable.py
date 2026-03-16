import code
import importlib
import typing

# SEC-002: exec
code_str = "print('hello')"
exec(code_str)
