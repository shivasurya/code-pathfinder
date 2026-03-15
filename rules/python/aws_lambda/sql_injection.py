"""
Python AWS Lambda SQL Injection Rules

PYTHON-LAMBDA-SEC-010: SQL Injection via MySQL cursor
PYTHON-LAMBDA-SEC-011: SQL Injection via psycopg2 cursor
PYTHON-LAMBDA-SEC-012: SQL Injection via pymssql cursor
PYTHON-LAMBDA-SEC-013: SQL Injection via PyMySQL cursor
PYTHON-LAMBDA-SEC-014: SQL Injection via SQLAlchemy session
PYTHON-LAMBDA-SEC-015: Tainted SQL String Construction
PYTHON-LAMBDA-SEC-016: DynamoDB Filter Injection

Security Impact: CRITICAL
CWE: CWE-89 (Improper Neutralization of Special Elements used in an SQL Command)
OWASP: A03:2021 - Injection

DESCRIPTION:
These rules detect SQL injection vulnerabilities in AWS Lambda functions where untrusted
event data flows into database query execution. Lambda functions commonly interact with
RDS, Aurora, or DynamoDB, and event data from API Gateway, S3, SNS, or other triggers
is frequently used to construct queries without proper parameterization.

Detected database sinks:
- **MySQL connector**: cursor.execute() / cursor.executemany() with tainted query strings
- **psycopg2 (PostgreSQL)**: cursor.execute(), cursor.executemany(), cursor.mogrify()
- **pymssql (SQL Server)**: cursor.execute() with string-formatted queries
- **PyMySQL**: cursor.execute() / cursor.executemany() with tainted input
- **SQLAlchemy**: session.execute() with raw SQL strings instead of parameterized text()
- **String construction**: f-strings, .format(), or % formatting used to build SQL queries
- **DynamoDB**: Tainted filter expressions in scan() and query() operations (NoSQL injection)

SECURITY IMPLICATIONS:

**1. Data Breach**:
SQL injection allows attackers to extract entire database contents including user credentials,
personal information, financial records, and other sensitive data using UNION-based or
blind injection techniques.

**2. Authentication Bypass**:
Attackers can modify WHERE clauses to bypass login queries (e.g., ' OR '1'='1), gaining
unauthorized access to any account.

**3. Data Manipulation**:
INSERT, UPDATE, and DELETE statements can be injected to modify or destroy data,
potentially causing data integrity issues or complete data loss.

**4. Privilege Escalation**:
In some database configurations, SQL injection can lead to operating system command
execution (e.g., xp_cmdshell in SQL Server, COPY TO PROGRAM in PostgreSQL).

**5. DynamoDB NoSQL Injection**:
Tainted filter expressions in DynamoDB scan/query operations can allow attackers to
modify query conditions and access unauthorized data.

VULNERABLE EXAMPLE:
```python
import pymysql

def lambda_handler(event, context):
    user_id = event.get('user_id', '')
    conn = pymysql.connect(host='rds-host', user='admin', password='pass', db='mydb')
    cursor = conn.cursor()

    # VULNERABLE: String formatting in SQL query
    query = f"SELECT * FROM users WHERE id = '{user_id}'"
    cursor.execute(query)  # SQL injection!

    # VULNERABLE: .format() in SQL query
    query = "SELECT * FROM orders WHERE user_id = {}".format(user_id)
    cursor.execute(query)

    # VULNERABLE: String concatenation
    query = "DELETE FROM logs WHERE date < '" + event.get('date') + "'"
    cursor.execute(query)

    return {'statusCode': 200, 'body': cursor.fetchall()}

# Attack payload:
# event = {"user_id": "' OR '1'='1' UNION SELECT username,password FROM admin_users--"}
```

SECURE EXAMPLE:
```python
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
```

DETECTION AND PREVENTION:
```bash
# Scan for Lambda SQL injection
pathfinder scan --project . --ruleset cpf/python/PYTHON-LAMBDA-SEC-010

# CI/CD integration
- name: Check Lambda SQL injection
  run: pathfinder ci --project . --ruleset cpf/python/aws_lambda
```

**Code Review Checklist**:
- [ ] All SQL queries use parameterized placeholders (%s, :param), never string formatting
- [ ] No f-strings, .format(), or % operator used to build SQL queries
- [ ] SQLAlchemy raw queries use text() with bindparams()
- [ ] ORM query methods are preferred over raw SQL
- [ ] Input validation (type checking, allowlists) applied before database operations
- [ ] DynamoDB operations use boto3 SDK expressions, not raw filter strings
- [ ] Database user has minimal required privileges (no DROP, no GRANT)

COMPLIANCE:
- OWASP A03:2021: Injection
- CWE-89: SQL Injection
- CWE-943: Improper Neutralization of Special Elements in Data Query Logic (DynamoDB)
- PCI DSS Requirement 6.5.1: Injection flaws
- SANS Top 25: CWE-89 consistently ranked in top 5

REFERENCES:
- CWE-89: SQL Injection (https://cwe.mitre.org/data/definitions/89.html)
- CWE-943: NoSQL Injection (https://cwe.mitre.org/data/definitions/943.html)
- OWASP SQL Injection Prevention Cheat Sheet (https://cheatsheetseries.owasp.org/cheatsheets/SQL_Injection_Prevention_Cheat_Sheet.html)
- AWS Lambda with RDS Best Practices (https://docs.aws.amazon.com/lambda/latest/dg/services-rds.html)
- OWASP Injection (https://owasp.org/Top10/A03_2021-Injection/)
"""

from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets


class DBCursor(QueryType):
    fqns = ["sqlite3.Cursor", "mysql.connector.cursor.MySQLCursor",
            "psycopg2.extensions.cursor", "pymysql.cursors.Cursor"]
    patterns = ["*Cursor"]
    match_subclasses = True


class SQLAlchemySession(QueryType):
    fqns = ["sqlalchemy.orm.Session", "sqlalchemy.orm.session.Session"]
    patterns = ["*Session"]
    match_subclasses = True


_LAMBDA_SOURCES = [
    calls("event.get"),
    calls("event.items"),
    calls("event.values"),
    calls("*.get"),
]


@python_rule(
    id="PYTHON-LAMBDA-SEC-010",
    name="Lambda SQL Injection via MySQL Cursor",
    severity="CRITICAL",
    category="aws_lambda",
    cwe="CWE-89",
    tags="python,aws,lambda,sql-injection,mysql,owasp-a03,cwe-89",
    message="Lambda event data flows to MySQL cursor.execute(). Use parameterized queries.",
    owasp="A03:2021",
)
def detect_lambda_mysql_sqli():
    """Detects Lambda event data flowing to MySQL cursor.execute()."""
    return flows(
        from_sources=_LAMBDA_SOURCES,
        to_sinks=[
            DBCursor.method("execute", "executemany").tracks(0),
        ],
        sanitized_by=[
            calls("escape"),
            calls("escape_string"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )


@python_rule(
    id="PYTHON-LAMBDA-SEC-011",
    name="Lambda SQL Injection via psycopg2 Cursor",
    severity="CRITICAL",
    category="aws_lambda",
    cwe="CWE-89",
    tags="python,aws,lambda,sql-injection,psycopg2,owasp-a03,cwe-89",
    message="Lambda event data flows to psycopg2 cursor.execute(). Use parameterized queries.",
    owasp="A03:2021",
)
def detect_lambda_psycopg2_sqli():
    """Detects Lambda event data flowing to psycopg2 cursor."""
    return flows(
        from_sources=_LAMBDA_SOURCES,
        to_sinks=[
            DBCursor.method("execute", "executemany", "mogrify").tracks(0),
            calls("cursor.execute"),
            calls("cursor.mogrify"),
        ],
        sanitized_by=[
            calls("escape"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )


@python_rule(
    id="PYTHON-LAMBDA-SEC-012",
    name="Lambda SQL Injection via pymssql Cursor",
    severity="CRITICAL",
    category="aws_lambda",
    cwe="CWE-89",
    tags="python,aws,lambda,sql-injection,pymssql,owasp-a03,cwe-89",
    message="Lambda event data flows to pymssql cursor.execute(). Use parameterized queries.",
    owasp="A03:2021",
)
def detect_lambda_pymssql_sqli():
    """Detects Lambda event data flowing to pymssql cursor."""
    return flows(
        from_sources=_LAMBDA_SOURCES,
        to_sinks=[
            DBCursor.method("execute").tracks(0),
            calls("cursor.execute"),
        ],
        sanitized_by=[
            calls("escape"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )


@python_rule(
    id="PYTHON-LAMBDA-SEC-013",
    name="Lambda SQL Injection via PyMySQL Cursor",
    severity="CRITICAL",
    category="aws_lambda",
    cwe="CWE-89",
    tags="python,aws,lambda,sql-injection,pymysql,owasp-a03,cwe-89",
    message="Lambda event data flows to PyMySQL cursor.execute(). Use parameterized queries.",
    owasp="A03:2021",
)
def detect_lambda_pymysql_sqli():
    """Detects Lambda event data flowing to PyMySQL cursor."""
    return flows(
        from_sources=_LAMBDA_SOURCES,
        to_sinks=[
            DBCursor.method("execute", "executemany").tracks(0),
            calls("cursor.execute"),
        ],
        sanitized_by=[
            calls("escape"),
            calls("escape_string"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )


@python_rule(
    id="PYTHON-LAMBDA-SEC-014",
    name="Lambda SQL Injection via SQLAlchemy",
    severity="CRITICAL",
    category="aws_lambda",
    cwe="CWE-89",
    tags="python,aws,lambda,sql-injection,sqlalchemy,owasp-a03,cwe-89",
    message="Lambda event data flows to SQLAlchemy session.execute(). Use text() with params.",
    owasp="A03:2021",
)
def detect_lambda_sqlalchemy_sqli():
    """Detects Lambda event data flowing to SQLAlchemy session.execute()."""
    return flows(
        from_sources=_LAMBDA_SOURCES,
        to_sinks=[
            SQLAlchemySession.method("execute").tracks(0),
            calls("session.execute"),
            calls("*.execute"),
        ],
        sanitized_by=[
            calls("text"),
            calls("sqlalchemy.text"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )


@python_rule(
    id="PYTHON-LAMBDA-SEC-015",
    name="Lambda Tainted SQL String Construction",
    severity="HIGH",
    category="aws_lambda",
    cwe="CWE-89",
    tags="python,aws,lambda,sql-injection,string-format,owasp-a03,cwe-89",
    message="Lambda event data used in SQL string construction. Use parameterized queries.",
    owasp="A03:2021",
)
def detect_lambda_tainted_sql():
    """Detects Lambda event data in string formatting that reaches execute()."""
    return flows(
        from_sources=_LAMBDA_SOURCES,
        to_sinks=[
            calls("cursor.execute"),
            calls("*.execute"),
        ],
        sanitized_by=[
            calls("escape"),
            calls("int"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )


@python_rule(
    id="PYTHON-LAMBDA-SEC-016",
    name="Lambda DynamoDB Filter Injection",
    severity="HIGH",
    category="aws_lambda",
    cwe="CWE-943",
    tags="python,aws,lambda,dynamodb,nosql-injection,owasp-a03,cwe-943",
    message="Lambda event data flows to DynamoDB scan/query filter. Validate input.",
    owasp="A03:2021",
)
def detect_lambda_dynamodb_injection():
    """Detects Lambda event data flowing to DynamoDB scan/query filters."""
    return flows(
        from_sources=_LAMBDA_SOURCES,
        to_sinks=[
            calls("*.scan"),
            calls("*.query"),
            calls("table.scan"),
            calls("table.query"),
        ],
        sanitized_by=[],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
