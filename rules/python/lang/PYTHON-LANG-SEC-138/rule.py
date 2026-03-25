from rules.python_decorators import python_rule
from codepathfinder import calls, QueryType, Or


class Neo4jModule(QueryType):
    fqns = ["neo4j"]


@python_rule(
    id="PYTHON-LANG-SEC-138",
    name="Cypher/Graph Query Injection via String Concatenation",
    severity="HIGH",
    category="lang",
    cwe="CWE-943",
    tags="python,neo4j,cypher,injection,graph-database,OWASP-A03,CWE-943",
    message="Cypher query built with string formatting detected. Use parameterized queries with $param syntax instead. See GHSA-gg5m-55jj-8m5g.",
    owasp="A03:2021",
)
def detect_cypher_injection():
    """Detects graph database query execution methods that may receive string-formatted queries.
    CVE: GHSA-gg5m-55jj-8m5g (graphiti-core Cypher injection via node_labels).

    Uses type-inferred Neo4jModule for neo4j driver calls, plus targeted calls()
    for py2neo and general graph DB patterns. Avoids overly broad *.run which
    would match subprocess.run, asyncio.run, etc.
    """
    return Or(
        Neo4jModule.method("execute_query"),
        calls("*.cypher.execute"),
        calls("*.evaluate"),
    )
