import os
import subprocess
import asyncio

# SEC-003: os.spawn with event data
def handler_spawn(event, context):
    prog = event.get('program')
    os.spawnl(os.P_NOWAIT, prog, prog)
    return {"statusCode": 200}
