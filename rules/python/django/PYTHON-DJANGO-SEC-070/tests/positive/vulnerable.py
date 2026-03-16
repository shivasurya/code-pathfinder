import pickle
import yaml
from django.views.decorators.csrf import csrf_exempt


# SEC-070: insecure cookies
def set_insecure_cookie(request):
    response = HttpResponse("OK")
    response.set_cookie("session_id", "abc123")
    return response


# SEC-071: CSRF exempt
@csrf_exempt
def unprotected_view(request):
    return HttpResponse("No CSRF check")


# SEC-072: insecure deserialization
def vulnerable_pickle(request):
    data = request.POST.get('data')
    obj = pickle.loads(data)
    return obj


def vulnerable_yaml(request):
    content = request.POST.get('config')
    obj = yaml.load(content)
    return obj
