import os
import subprocess
import asyncio

# SEC-006: loop.subprocess_exec
async def handler_loop_exec(event, context):
    prog = event.get('program')
    loop = asyncio.get_event_loop()
    transport, protocol = await loop.subprocess_exec(lambda: None, prog)
    return {"statusCode": 200}
