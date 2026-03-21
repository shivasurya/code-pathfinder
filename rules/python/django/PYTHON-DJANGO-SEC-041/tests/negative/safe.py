from django.http import HttpResponse, FileResponse, Http404
import os

UPLOAD_DIR = '/var/www/uploads/'

def download_file(request):
    # SECURE: Validate resolved path stays within allowed directory
    filename = request.GET.get('file', '')
    # Use basename to strip directory traversal sequences
    safe_name = os.path.basename(filename)
    filepath = os.path.join(UPLOAD_DIR, safe_name)
    # Double-check with realpath
    real_path = os.path.realpath(filepath)
    if not real_path.startswith(os.path.realpath(UPLOAD_DIR)):
        raise Http404("File not found")
    if not os.path.isfile(real_path):
        raise Http404("File not found")
    return FileResponse(open(real_path, 'rb'))

def read_template(request):
    # SECURE: Allowlist of permitted templates
    ALLOWED_TEMPLATES = {'header.html', 'footer.html', 'sidebar.html'}
    template = request.GET.get('template', '')
    if template not in ALLOWED_TEMPLATES:
        raise Http404("Template not found")
    filepath = os.path.join('/app/templates/', template)
    with open(filepath) as f:
        return HttpResponse(f.read())
