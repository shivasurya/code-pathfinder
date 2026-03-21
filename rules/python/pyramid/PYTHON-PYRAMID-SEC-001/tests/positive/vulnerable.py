from pyramid.config import Configurator
from pyramid.response import Response

# SEC-001: CSRF disabled
config = Configurator()
config.set_default_csrf_options(require_csrf=False)
