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
