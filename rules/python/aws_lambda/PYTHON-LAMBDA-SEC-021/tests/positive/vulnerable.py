def handler(event, context):
    name = event.get('name')
    html = "<html><body><h1>Hello " + name + "</h1></body></html>"
    return {
        'statusCode': 200,
        'headers': {'Content-Type': 'text/html'},
        'body': html
    }
