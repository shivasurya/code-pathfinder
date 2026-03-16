from django.db import connection
from django.db.models.expressions import RawSQL
from django.http import HttpRequest

# SEC-002: ORM .raw() with request data
def vulnerable_raw(request):
    name = request.POST.get('name')
    users = User.objects.raw(f"SELECT * FROM users WHERE name = '{name}'")
    return users
