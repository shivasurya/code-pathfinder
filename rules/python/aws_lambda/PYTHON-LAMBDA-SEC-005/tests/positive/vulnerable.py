import os
import subprocess
import asyncio


# SEC-001: os.system with event data
def handler_os_system(event, context):
    filename = event.get('filename')
    os.system(f"cat {filename}")
    return {"statusCode": 200}


# SEC-002: subprocess with event data
def handler_subprocess(event, context):
    cmd = event.get('command')
    result = subprocess.call(cmd, shell=True)
    return {"statusCode": 200, "body": str(result)}


def handler_subprocess_popen(event, context):
    host = event.get('host')
    proc = subprocess.Popen(f"ping {host}", shell=True)
    return {"statusCode": 200}


# SEC-003: os.spawn with event data
def handler_spawn(event, context):
    prog = event.get('program')
    os.spawnl(os.P_NOWAIT, prog, prog)
    return {"statusCode": 200}


# SEC-004: asyncio shell
async def handler_asyncio_shell(event, context):
    cmd = event.get('cmd')
    proc = await asyncio.create_subprocess_shell(cmd)
    return {"statusCode": 200}


# SEC-005: asyncio exec
async def handler_asyncio_exec(event, context):
    prog = event.get('program')
    proc = await asyncio.create_subprocess_exec(prog)
    return {"statusCode": 200}


# SEC-006: loop.subprocess_exec
async def handler_loop_exec(event, context):
    prog = event.get('program')
    loop = asyncio.get_event_loop()
    transport, protocol = await loop.subprocess_exec(lambda: None, prog)
    return {"statusCode": 200}
