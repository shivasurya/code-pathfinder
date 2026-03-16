from pyramid.config import Configurator
from pyramid.response import Response


# SEC-001: CSRF disabled
config = Configurator()
config.set_default_csrf_options(require_csrf=False)


# SEC-002: Direct response XSS
def vulnerable_view(request):
    name = request.params.get('name')
    return Response(f"Hello {name}")


# SEC-003: SQLAlchemy SQL injection
def vulnerable_query(request):
    search = request.params.get('q')
    results = session.query(User).filter(f"name = '{search}'")
    return results


def vulnerable_order(request):
    sort_col = request.params.get('sort')
    results = session.query(Item).order_by(sort_col)
    return results
