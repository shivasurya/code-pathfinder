import json

# SEC-010: MySQL cursor.execute
def handler_mysql(event, context):
    import mysql.connector
    conn = mysql.connector.connect(host="db", database="app")
    cursor = conn.cursor()
    user_id = event.get('user_id')
    cursor.execute(f"SELECT * FROM users WHERE id = {user_id}")
    return {"statusCode": 200, "body": json.dumps(cursor.fetchall())}
