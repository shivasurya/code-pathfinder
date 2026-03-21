import subprocess
import shlex
import os

def lambda_handler(event, context):
    # SECURE: Use subprocess with list arguments (no shell interpretation)
    filename = event.get('filename', '')
    result = subprocess.run(
        ['ls', '-la', f'/tmp/{filename}'],  # List args = no shell injection
        capture_output=True, text=True
    )

    # SECURE: Use shlex.quote() for shell commands when unavoidable
    user_input = event.get('name', '')
    safe_input = shlex.quote(user_input)
    result = subprocess.run(
        f'echo {safe_input}',
        shell=True, capture_output=True, text=True
    )

    # SECURE: Validate and sanitize input
    import re
    filename = event.get('filename', '')
    if not re.match(r'^[a-zA-Z0-9._-]+$', filename):
        return {'statusCode': 400, 'body': 'Invalid filename'}

    result = subprocess.run(
        ['cat', f'/tmp/{filename}'],
        capture_output=True, text=True
    )

    return {'statusCode': 200, 'body': result.stdout}
