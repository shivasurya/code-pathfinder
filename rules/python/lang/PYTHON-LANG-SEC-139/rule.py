from rules.python_decorators import python_rule
from codepathfinder import calls, QueryType


class MsgpackModule(QueryType):
    fqns = ["msgpack", "ormsgpack"]


@python_rule(
    id="PYTHON-LANG-SEC-139",
    name="Unsafe msgpack Deserialization",
    severity="HIGH",
    category="lang",
    cwe="CWE-502",
    tags="python,msgpack,deserialization,untrusted-data,OWASP-A08,CWE-502",
    message="msgpack.unpackb() or msgpack.unpack() detected. Deserializing untrusted msgpack data with ext_hook can execute arbitrary code.",
    owasp="A08:2021",
)
def detect_msgpack_deserialization():
    """Detects msgpack.unpackb(), msgpack.unpack(), and msgpack.Unpacker() calls."""
    return MsgpackModule.method("unpackb", "unpack", "Unpacker")
