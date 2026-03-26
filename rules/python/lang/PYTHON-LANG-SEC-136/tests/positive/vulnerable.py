import importlib
from scrapy.utils.misc import load_object

# SEC-136: User input flows to dynamic module import (single-file patterns)

# 1. Flask request header → importlib.import_module (Scrapy CVE pattern)
def load_handler_from_header(request):
    handler_module = request.headers.get("X-Handler-Module")
    mod = importlib.import_module(handler_module)
    return mod.handle(request)

# 2. Flask request form → load_object
def load_serializer(request):
    serializer_path = request.form.get("serializer")
    serializer_cls = load_object(serializer_path)
    return serializer_cls()

# 3. Flask request args → __import__
def load_plugin_from_query(request):
    plugin_name = request.args.get("plugin")
    mod = __import__(plugin_name)
    return mod
