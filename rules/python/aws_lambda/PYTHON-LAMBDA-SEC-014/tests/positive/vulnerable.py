import json

# SEC-014: SQLAlchemy session.execute
def handler_sqlalchemy(event, context):
    search = event.get('search')
    result = session.execute(f"SELECT * FROM items WHERE name = '{search}'")
    return {"statusCode": 200}
