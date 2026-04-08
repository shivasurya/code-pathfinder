from neo4j import GraphDatabase

driver = GraphDatabase.driver("bolt://localhost:7687", auth=("neo4j", "password"))

# SEC-138: Cypher injection via f-string in session.run()
label = user_input
with driver.session() as session:
    result = session.run(f"MATCH (n:{label}) RETURN n")

# SEC-138: Cypher injection via string concatenation in tx.run()
def find_user(tx, name):
    query = "MATCH (u:User) WHERE u.name = '" + name + "' RETURN u"
    return tx.run(query)

# SEC-138: Cypher injection via format() in execute_query()
user_query = input("Enter search term: ")
driver.execute_query("MATCH (n) WHERE n.name = '%s' RETURN n" % user_query)

# SEC-138: Cypher injection via graph.evaluate()
from py2neo import Graph
graph = Graph("bolt://localhost:7687")
graph.evaluate("MATCH (n) WHERE n.id = " + user_id + " RETURN n")

# SEC-138: Cypher injection via cypher.execute()
db.cypher.execute("CREATE (n:User {name: '" + username + "'})")
