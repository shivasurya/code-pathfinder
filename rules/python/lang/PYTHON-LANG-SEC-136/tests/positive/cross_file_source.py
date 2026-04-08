# SEC-136: Cross-file source — user input extracted here, flows to sink in cross_file_sink.py

def get_module_from_request(request):
    """Extract module path from user request — this is the taint source."""
    module_path = request.args.get("module")
    return module_path


def get_backend_from_config(config):
    """Extract backend path from config — config.get() as source."""
    backend = config.get("AUTH_BACKEND")
    return backend
