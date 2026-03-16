from django.http import HttpResponse, JsonResponse
from django.middleware.csrf import CsrfViewMiddleware
import json

def set_preferences(request):
    # SECURE: Cookie set with all security flags
    response = HttpResponse("Preferences saved")
    response.set_cookie(
        'session_id',
        request.session.session_key,
        secure=True,      # Only sent over HTTPS
        httponly=True,     # Not accessible via JavaScript
        samesite='Lax',   # Prevents cross-site sending
        max_age=3600,     # 1 hour expiry
    )
    return response

# SECURE: No @csrf_exempt - CSRF protection enabled by default
def transfer_funds(request):
    if request.method != 'POST':
        return HttpResponse(status=405)
    amount = request.POST.get('amount')
    recipient = request.POST.get('recipient')
    perform_transfer(request.user, recipient, amount)
    return HttpResponse("Transfer complete")

def load_data(request):
    # SECURE: Use JSON instead of pickle for untrusted data
    try:
        data = json.loads(request.body)
    except json.JSONDecodeError:
        return JsonResponse({'error': 'Invalid JSON'}, status=400)
    return JsonResponse({'data': data})
