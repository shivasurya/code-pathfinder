import json

# SEC-016: DynamoDB filter injection
def handler_dynamodb(event, context):
    import boto3
    table = boto3.resource('dynamodb').Table('users')
    filter_expr = event.get('filter')
    result = table.scan(FilterExpression=filter_expr)
    return {"statusCode": 200, "body": json.dumps(result)}
