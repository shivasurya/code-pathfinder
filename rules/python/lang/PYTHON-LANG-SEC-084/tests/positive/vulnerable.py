import sqlite3

# SEC-084: formatted SQL (general)
conn2 = sqlite3.connect("test.db")
cursor = conn2.cursor()
cursor.execute("SELECT * FROM products WHERE id = " + product_id)
