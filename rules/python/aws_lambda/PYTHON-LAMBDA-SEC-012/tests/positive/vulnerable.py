import pymssql

def handler(event, context):
    user_id = event.get('user_id')
    conn = pymssql.connect(server='db', user='sa', password='pass', database='app')
    cursor = conn.cursor()
    cursor.execute("SELECT * FROM users WHERE id = '" + user_id + "'")
    return cursor.fetchall()
