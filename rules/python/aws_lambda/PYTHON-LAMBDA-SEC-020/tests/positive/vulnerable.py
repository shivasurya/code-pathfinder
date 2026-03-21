import json
import pickle

# SEC-020: tainted HTML response
def handler_html_response(event, context):
    name = event.get('name')
    body = f"<html><body>Hello {name}</body></html>"
    return {
        "statusCode": 200,
        "body": json.dumps({"html": body}),
        "headers": {"Content-Type": "text/html"}
    }
