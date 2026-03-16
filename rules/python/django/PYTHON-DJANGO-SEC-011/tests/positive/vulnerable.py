import os
import subprocess

    filename = request.GET.get('file')
    os.system(f"cat {filename}")


    cmd = request.POST.get('command')
    subprocess.call(cmd, shell=True)


    host = request.GET.get('host')
    proc = subprocess.Popen(f"ping {host}", shell=True)
    return proc.communicate()
