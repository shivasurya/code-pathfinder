import code
import importlib
import typing

# SEC-005: non-literal import
module_name = "os"
mod = __import__(module_name)
mod2 = importlib.import_module(module_name)
