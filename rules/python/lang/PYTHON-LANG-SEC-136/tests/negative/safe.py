# Safe: Standard import statements (not dynamic imports)
import json
import os.path
import collections

# Safe: Using importlib.resources (not import_module)
def read_package_data():
    import importlib.resources
    return importlib.resources.read_text("mypackage", "data.txt")

# Safe: Using importlib.metadata (not import_module)
def get_version():
    from importlib.metadata import version
    return version("mypackage")

# Safe: Using importlib.metadata entry_points (not import_module)
def load_plugins_via_entrypoints():
    from importlib.metadata import entry_points
    eps = entry_points(group="myapp.plugins")
    return {ep.name: ep.load() for ep in eps}

# Safe: Regular function calls that happen to parse strings
def parse_data(raw):
    return json.loads(raw)

# Safe: Using pkgutil (not import_module or __import__)
def list_packages():
    import pkgutil
    return [mod.name for mod in pkgutil.iter_modules()]
