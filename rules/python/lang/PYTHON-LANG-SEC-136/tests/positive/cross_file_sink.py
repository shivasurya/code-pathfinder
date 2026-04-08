import importlib
from cross_file_source import get_module_from_request, get_backend_from_config

# SEC-136: Cross-file sink — tainted value from cross_file_source.py reaches import here

def load_user_module(request):
    """Cross-file flow: request.args.get("module") → importlib.import_module()"""
    module_path = get_module_from_request(request)
    mod = importlib.import_module(module_path)
    return mod


def load_auth_backend(config):
    """Cross-file flow: config.get("AUTH_BACKEND") → importlib.import_module()"""
    backend_path = get_backend_from_config(config)
    mod = importlib.import_module(backend_path)
    return getattr(mod, "Backend")
