import subprocess
import asyncio

# SEC-021: subprocess with shell=True
subprocess.call("rm -rf /tmp/*", shell=True)
subprocess.run("ls", shell=True)
