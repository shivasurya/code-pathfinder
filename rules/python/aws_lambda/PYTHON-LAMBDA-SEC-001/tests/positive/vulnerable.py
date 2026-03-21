import os
import subprocess
import asyncio

# SEC-001: os.system with event data
def handler_os_system(event, context):
    filename = event.get('filename')
    os.system(f"cat {filename}")
    return {"statusCode": 200}
