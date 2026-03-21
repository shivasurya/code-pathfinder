import json

# SEC-015: tainted SQL string
def handler_tainted_sql(event, context):
    table = event.get('table')
    query = "SELECT * FROM " + table
    cursor.execute(query)
    return {"statusCode": 200}
