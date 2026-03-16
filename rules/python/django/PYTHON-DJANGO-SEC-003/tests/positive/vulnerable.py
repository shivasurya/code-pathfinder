from django.db import connection
from django.db.models.expressions import RawSQL
from django.http import HttpRequest


# SEC-001: cursor.execute with request data
def vulnerable_cursor(request):
    user_id = request.GET.get('id')
    cursor = connection.cursor()
    query = f"SELECT * FROM users WHERE id = {user_id}"
    cursor.execute(query)
    return cursor.fetchone()


# SEC-002: ORM .raw() with request data
def vulnerable_raw(request):
    name = request.POST.get('name')
    users = User.objects.raw(f"SELECT * FROM users WHERE name = '{name}'")
    return users


# SEC-003: ORM .extra() with request data
def vulnerable_extra(request):
    where_clause = request.GET.get('filter')
    results = Article.objects.extra(where=[where_clause])
    return results


# SEC-004: RawSQL expression with request data
def vulnerable_rawsql(request):
    order = request.GET.get('order')
    expr = RawSQL(f"SELECT * FROM products ORDER BY {order}", [])
    return expr


# SEC-005: Raw SQL usage (audit)
def audit_rawsql():
    expr = RawSQL("SELECT 1", [])
    return expr


# SEC-006: Tainted SQL string (same as SEC-001 pattern)
def vulnerable_tainted_sql(request):
    search = request.GET.get('q')
    query = "SELECT * FROM items WHERE name LIKE '%" + search + "%'"
    cursor = connection.cursor()
    cursor.execute(query)
    return cursor.fetchall()
