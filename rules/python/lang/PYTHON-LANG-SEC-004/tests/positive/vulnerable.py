import code
import importlib
import typing

# SEC-004: globals
def render(template, **kwargs):
    return template.format(**globals())
