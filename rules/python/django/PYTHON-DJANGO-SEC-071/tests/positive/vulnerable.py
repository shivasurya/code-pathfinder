import pickle
import yaml
from django.views.decorators.csrf import csrf_exempt

# SEC-071: CSRF exempt
@csrf_exempt
def unprotected_view(request):
    return HttpResponse("No CSRF check")
