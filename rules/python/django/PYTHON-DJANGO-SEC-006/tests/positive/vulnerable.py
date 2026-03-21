from django.db import connection
from django.db.models.expressions import RawSQL
from django.http import HttpRequest

# SEC-006: Tainted SQL string (same as SEC-001 pattern)
def vulnerable_tainted_sql(request):
    search = request.GET.get('q')
    query = "SELECT * FROM items WHERE name LIKE '%" + search + "%'"
    cursor = connection.cursor()
    cursor.execute(query)
    return cursor.fetchall()
