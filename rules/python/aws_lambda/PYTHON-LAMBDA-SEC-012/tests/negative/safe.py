import pymysql
from sqlalchemy import text

def lambda_handler(event, context):
    user_id = event.get('user_id', '')
    conn = pymysql.connect(host='rds-host', user='admin', password='pass', db='mydb')
    cursor = conn.cursor()

    # SECURE: Parameterized query with placeholders
    query = "SELECT * FROM users WHERE id = %s"
    cursor.execute(query, (user_id,))

    # SECURE: SQLAlchemy with text() and bindparams
    stmt = text("SELECT * FROM orders WHERE user_id = :uid").bindparams(uid=user_id)
    session.execute(stmt)

    # SECURE: Input validation before query
    try:
        user_id_int = int(user_id)
    except ValueError:
        return {'statusCode': 400, 'body': 'Invalid user ID'}
    cursor.execute("SELECT * FROM users WHERE id = %s", (user_id_int,))

    # SECURE: DynamoDB with proper Key/FilterExpression
    import boto3
    table = boto3.resource('dynamodb').Table('users')
    response = table.query(
        KeyConditionExpression=Key('user_id').eq(user_id)  # SDK handles escaping
    )

    return {'statusCode': 200, 'body': cursor.fetchall()}
