import os

# SEC-040: path traversal via open
def vulnerable_open(request):
    filename = request.GET.get('file')
    with open(filename) as f:
        return f.read()


# SEC-041: path traversal via os.path.join
def vulnerable_path_join(request):
    user_path = request.GET.get('path')
    full_path = os.path.join('/uploads', user_path)
    with open(full_path) as f:
        return f.read()
