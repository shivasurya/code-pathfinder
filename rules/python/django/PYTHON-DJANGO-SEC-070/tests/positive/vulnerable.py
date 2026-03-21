import pickle
import yaml
from django.views.decorators.csrf import csrf_exempt

# SEC-070: insecure cookies
def set_insecure_cookie(request):
    response = HttpResponse("OK")
    response.set_cookie("session_id", "abc123")
    return response
