from pyramid.config import Configurator
from pyramid.response import Response

# SEC-002: Direct response XSS
def vulnerable_view(request):
    name = request.params.get('name')
    return Response(f"Hello {name}")
