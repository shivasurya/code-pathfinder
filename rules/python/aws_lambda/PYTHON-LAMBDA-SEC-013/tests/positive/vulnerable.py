import pymysql

def handler(event, context):
    search = event.get('search')
    conn = pymysql.connect(host='db', user='root', password='pass', database='app')
    cursor = conn.cursor()
    cursor.execute("SELECT * FROM products WHERE name LIKE '%" + search + "%'")
    return cursor.fetchall()
