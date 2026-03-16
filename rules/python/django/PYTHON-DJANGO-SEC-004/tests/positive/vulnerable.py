from django.db import connection
from django.db.models.expressions import RawSQL
from django.http import HttpRequest

# SEC-004: RawSQL expression with request data
def vulnerable_rawsql(request):
    order = request.GET.get('order')
    expr = RawSQL(f"SELECT * FROM products ORDER BY {order}", [])
    return expr
