import json
import html
import ast

def lambda_handler(event, context):
    # SECURE: HTML-escape user input before embedding in response
    name = html.escape(event.get('name', ''))
    body = f'<html><body><h1>Hello, {name}!</h1></body></html>'
    return {
        'statusCode': 200,
        'headers': {'Content-Type': 'text/html'},
        'body': body
    }

def process_handler(event, context):
    # SECURE: Use ast.literal_eval() for safe expression parsing
    expression = event.get('calc', '')
    try:
        result = ast.literal_eval(expression)  # Only parses literals
    except (ValueError, SyntaxError):
        return {'statusCode': 400, 'body': 'Invalid expression'}

    # SECURE: Use JSON instead of pickle for deserialization
    data = event.get('data', '{}')
    try:
        obj = json.loads(data)  # Safe - no code execution
    except json.JSONDecodeError:
        return {'statusCode': 400, 'body': 'Invalid JSON'}

    # SECURE: Use allowlist for operations instead of eval/exec
    ALLOWED_OPS = {'add': lambda a, b: a + b, 'sub': lambda a, b: a - b}
    op = event.get('operation', '')
    if op not in ALLOWED_OPS:
        return {'statusCode': 400, 'body': 'Invalid operation'}

    return {'statusCode': 200, 'body': json.dumps({'result': str(result)})}
