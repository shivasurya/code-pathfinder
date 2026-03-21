from django.http import JsonResponse
import subprocess
import shlex
import re

def ping_host(request):
    # SECURE: Validate input and use subprocess with list args
    host = request.GET.get('host', '')
    # Whitelist validation: only allow valid hostnames/IPs
    if not re.match(r'^[a-zA-Z0-9._-]+$', host):
        return JsonResponse({'error': 'Invalid host'}, status=400)
    result = subprocess.run(['ping', '-c', '3', host], capture_output=True, text=True)
    return JsonResponse({'result': result.stdout})

def run_tool(request):
    # SECURE: Use list arguments, never shell=True with user input
    filename = request.POST.get('filename', '')
    if not re.match(r'^[a-zA-Z0-9._-]+$', filename):
        return JsonResponse({'error': 'Invalid filename'}, status=400)
    result = subprocess.run(['file', filename], capture_output=True, text=True)
    return JsonResponse({'result': result.stdout})
