import os
import subprocess


# SEC-010: os.system with request data
def vulnerable_os_system(request):
    filename = request.GET.get('file')
    os.system(f"cat {filename}")


# SEC-011: subprocess with request data
def vulnerable_subprocess(request):
    cmd = request.POST.get('command')
    subprocess.call(cmd, shell=True)


def vulnerable_subprocess_popen(request):
    host = request.GET.get('host')
    proc = subprocess.Popen(f"ping {host}", shell=True)
    return proc.communicate()
