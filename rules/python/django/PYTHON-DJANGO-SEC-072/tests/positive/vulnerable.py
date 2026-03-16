import pickle
import yaml
from django.views.decorators.csrf import csrf_exempt

# SEC-072: insecure deserialization
def vulnerable_pickle(request):
    data = request.POST.get('data')
    obj = pickle.loads(data)
    return obj


def vulnerable_yaml(request):
    content = request.POST.get('config')
    obj = yaml.load(content)
    return obj
