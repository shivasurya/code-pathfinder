import json
import pickle

# SEC-022: exec with event data
def handler_exec(event, context):
    code = event.get('code')
    exec(code)
    return {"statusCode": 200}
