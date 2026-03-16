import pg8000

conn = pg8000.connect(user="postgres", database="mydb")
cursor = conn.cursor()
user_input = "admin"
cursor.execute("SELECT * FROM users WHERE name = '" + user_input + "'")
