from pyramid.config import Configurator
from pyramid.response import Response

# SEC-003: SQLAlchemy SQL injection
def vulnerable_query(request):
    search = request.params.get('q')
    results = session.query(User).filter(f"name = '{search}'")
    return results


def vulnerable_order(request):
    sort_col = request.params.get('sort')
    results = session.query(Item).order_by(sort_col)
    return results
