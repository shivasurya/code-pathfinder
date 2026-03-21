import subprocess
import asyncio

# SEC-020: subprocess calls
subprocess.call(["ls", "-la"])
subprocess.check_output("whoami", shell=True)
subprocess.Popen("cat /etc/passwd", shell=True)
subprocess.run("echo hello", shell=True)
