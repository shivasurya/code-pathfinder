from django.db import connection
from django.db.models.expressions import RawSQL
from django.http import HttpRequest

# SEC-005: Raw SQL usage (audit)
def audit_rawsql():
    expr = RawSQL("SELECT 1", [])
    return expr
