import os
import subprocess
import asyncio

# SEC-002: subprocess with event data
def handler_subprocess(event, context):
    cmd = event.get('command')
    result = subprocess.call(cmd, shell=True)
    return {"statusCode": 200, "body": str(result)}


def handler_subprocess_popen(event, context):
    host = event.get('host')
    proc = subprocess.Popen(f"ping {host}", shell=True)
    return {"statusCode": 200}
