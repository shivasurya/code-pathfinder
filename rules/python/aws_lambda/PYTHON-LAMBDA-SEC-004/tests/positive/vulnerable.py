import os
import subprocess
import asyncio

# SEC-004: asyncio shell
async def handler_asyncio_shell(event, context):
    cmd = event.get('cmd')
    proc = await asyncio.create_subprocess_shell(cmd)
    return {"statusCode": 200}
