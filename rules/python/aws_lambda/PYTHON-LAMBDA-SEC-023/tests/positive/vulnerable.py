import json
import pickle

# SEC-023: pickle deserialization
def handler_pickle(event, context):
    data = event.get('payload')
    obj = pickle.loads(data)
    return {"statusCode": 200, "body": json.dumps(str(obj))}
