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


# SEC-022: eval with event data
def handler_eval(event, context):
    expr = event.get('expression')
    result = eval(expr)
    return {"statusCode": 200, "body": json.dumps({"result": str(result)})}


# SEC-022: exec with event data
def handler_exec(event, context):
    code = event.get('code')
    exec(code)
    return {"statusCode": 200}


# SEC-023: pickle deserialization
def handler_pickle(event, context):
    data = event.get('payload')
    obj = pickle.loads(data)
    return {"statusCode": 200, "body": json.dumps(str(obj))}
