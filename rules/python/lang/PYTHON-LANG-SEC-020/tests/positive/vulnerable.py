import subprocess
import asyncio

# SEC-020: subprocess calls
subprocess.call(["ls", "-la"])
subprocess.check_output("whoami", shell=True)
subprocess.Popen("cat /etc/passwd", shell=True)
subprocess.run("echo hello", shell=True)

# SEC-021: subprocess with shell=True
subprocess.call("rm -rf /tmp/*", shell=True)
subprocess.run("ls", shell=True)

# SEC-022: asyncio shell
async def run_cmd():
    proc = await asyncio.create_subprocess_shell("ls -la")
    await proc.communicate()

# SEC-023: subinterpreters
import _xxsubinterpreters
_xxsubinterpreters.run_string("print('hello')")
