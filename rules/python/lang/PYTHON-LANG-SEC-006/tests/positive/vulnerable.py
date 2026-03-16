import code
import importlib
import typing

# SEC-006: dangerous annotations
class Foo:
    x: "eval('malicious')" = 1

hints = typing.get_type_hints(Foo)
