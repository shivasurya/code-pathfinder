import json

# SEC-011: psycopg2
def handler_psycopg2(event, context):
    import psycopg2
    conn = psycopg2.connect("dbname=app")
    cursor = conn.cursor()
    name = event.get('name')
    cursor.execute(f"SELECT * FROM users WHERE name = '{name}'")
    return {"statusCode": 200}
