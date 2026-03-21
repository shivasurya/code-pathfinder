from django.db import connection
from django.db.models.expressions import RawSQL
from django.http import HttpRequest

# SEC-003: ORM .extra() with request data
def vulnerable_extra(request):
    where_clause = request.GET.get('filter')
    results = Article.objects.extra(where=[where_clause])
    return results
