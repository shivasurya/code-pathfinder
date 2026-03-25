import importlib
from scrapy.utils.misc import load_object

# SEC-136: Dynamic module import from user-controlled input

# 1. importlib.import_module with user input
def load_plugin(plugin_name):
    """Load a plugin module by name from user configuration."""
    module = importlib.import_module(plugin_name)
    return module.Plugin()

# 2. importlib.import_module with HTTP header value
def load_handler_from_header(request):
    handler_module = request.headers.get("X-Handler-Module")
    mod = importlib.import_module(handler_module)
    return mod.handle(request)

# 3. load_object from Scrapy (used for Referrer-Policy injection CVE)
def configure_middleware(settings):
    policy_cls = settings.get("REFERRER_POLICY")
    return load_object(policy_cls)

# 4. __import__ with dynamic module name
def dynamic_import_legacy(module_name):
    mod = __import__(module_name)
    return mod

# 5. importlib.import_module from database/config
def load_backend_from_config(config):
    backend_path = config.get("AUTH_BACKEND", "myapp.auth.default")
    module_path, _, class_name = backend_path.rpartition(".")
    mod = importlib.import_module(module_path)
    return getattr(mod, class_name)

# 6. load_object with form data
def load_serializer(request):
    serializer_path = request.form.get("serializer")
    serializer_cls = load_object(serializer_path)
    return serializer_cls()
