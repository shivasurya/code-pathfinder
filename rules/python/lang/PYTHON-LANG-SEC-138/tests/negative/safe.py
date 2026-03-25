from neo4j import GraphDatabase
import json

driver = GraphDatabase.driver("bolt://localhost:7687", auth=("neo4j", "password"))

# Safe: no graph query execution methods called here
# The rule detects *.run(), *.execute_query(), *.evaluate(), *.query()
# These safe patterns avoid calling those methods entirely.

# Safe: using JSON for data interchange instead of graph queries
data = json.loads('{"name": "Alice"}')

# Safe: using driver info (no query execution)
info = driver.get_server_info()

# Safe: closing driver (not a query method)
driver.close()

# Safe: storing query string without executing
query_template = "MATCH (u:User) WHERE u.name = $name RETURN u"
params = {"name": "Alice"}
