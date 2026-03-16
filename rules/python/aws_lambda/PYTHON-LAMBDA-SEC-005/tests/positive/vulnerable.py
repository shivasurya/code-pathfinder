import os
import subprocess
import asyncio

# SEC-005: asyncio exec
async def handler_asyncio_exec(event, context):
    prog = event.get('program')
    proc = await asyncio.create_subprocess_exec(prog)
    return {"statusCode": 200}
