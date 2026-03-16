import json


# SEC-010: MySQL cursor.execute
def handler_mysql(event, context):
    import mysql.connector
    conn = mysql.connector.connect(host="db", database="app")
    cursor = conn.cursor()
    user_id = event.get('user_id')
    cursor.execute(f"SELECT * FROM users WHERE id = {user_id}")
    return {"statusCode": 200, "body": json.dumps(cursor.fetchall())}


# SEC-011: psycopg2
def handler_psycopg2(event, context):
    import psycopg2
    conn = psycopg2.connect("dbname=app")
    cursor = conn.cursor()
    name = event.get('name')
    cursor.execute(f"SELECT * FROM users WHERE name = '{name}'")
    return {"statusCode": 200}


# SEC-014: SQLAlchemy session.execute
def handler_sqlalchemy(event, context):
    search = event.get('search')
    result = session.execute(f"SELECT * FROM items WHERE name = '{search}'")
    return {"statusCode": 200}


# SEC-015: tainted SQL string
def handler_tainted_sql(event, context):
    table = event.get('table')
    query = "SELECT * FROM " + table
    cursor.execute(query)
    return {"statusCode": 200}


# SEC-016: DynamoDB filter injection
def handler_dynamodb(event, context):
    import boto3
    table = boto3.resource('dynamodb').Table('users')
    filter_expr = event.get('filter')
    result = table.scan(FilterExpression=filter_expr)
    return {"statusCode": 200, "body": json.dumps(result)}
