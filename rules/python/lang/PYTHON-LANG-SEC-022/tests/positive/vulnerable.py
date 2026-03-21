import subprocess
import asyncio

# SEC-022: asyncio shell
async def run_cmd():
    proc = await asyncio.create_subprocess_shell("ls -la")
    await proc.communicate()
