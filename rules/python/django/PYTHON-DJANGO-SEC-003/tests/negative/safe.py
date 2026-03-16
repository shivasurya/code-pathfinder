from django.http import JsonResponse
from django.db import connection

def search_users(request):
    # SECURE: Parameterized query with %s placeholder
    username = request.GET.get('username')
    cursor = connection.cursor()
    cursor.execute("SELECT * FROM users WHERE username = %s", [username])
    results = cursor.fetchall()
    return JsonResponse({'users': results})

def get_orders(request):
    # SECURE: Use Django ORM with safe filtering
    status = request.GET.get('status')
    orders = Order.objects.filter(status=status).values()
    return JsonResponse({'orders': list(orders)})

def get_orders_raw(request):
    # SECURE: Parameterized .raw() query
    status = request.GET.get('status')
    orders = Order.objects.raw("SELECT * FROM orders WHERE status = %s", [status])
    return JsonResponse({'orders': list(orders)})
