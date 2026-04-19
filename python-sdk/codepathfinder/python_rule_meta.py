# python_rule_meta.py — Companion metadata for Python rule writing.
# Read by scripts/generate_sdk_manifest.py to produce sdk-manifest.json.
#
# Unlike Go, Python rules don't use QueryType classes — they use calls(),
# variable(), and attribute() with pattern-based matching. This file documents
# the most security-relevant stdlib and third-party modules that rule authors
# typically target.
#
# Method signatures use Python syntax (def-style). The `fqns` field is the
# canonical import path used with calls(). For classes with instance methods,
# the FQN is the dotted path (e.g., "sqlite3.Cursor", "psycopg2.extensions.cursor").

SDK_META: dict = {

    # =====================================================================
    # Command execution
    # =====================================================================

    "PySubprocess": {
        "description": (
            "The subprocess standard library module for spawning child processes. "
            "Most call APIs accept either a list[str] (safe) or a string with shell=True "
            "(command-injection sink when the string contains user input)."
        ),
        "category": "command-execution",
        "fqns": ["subprocess"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "run": {
                "signature": "subprocess.run(args, *, shell=False, capture_output=False, ...) -> CompletedProcess",
                "description": "Runs a command and waits for completion. Sink when args is a string with shell=True.",
                "role": "sink",
                "tracks": [0],
            },
            "call": {
                "signature": "subprocess.call(args, *, shell=False, ...) -> int",
                "description": "Runs a command and returns its exit code. Sink under shell=True.",
                "role": "sink",
                "tracks": [0],
            },
            "check_call": {
                "signature": "subprocess.check_call(args, *, shell=False, ...) -> int",
                "description": "Like call() but raises on non-zero exit. Same injection risk.",
                "role": "sink",
                "tracks": [0],
            },
            "check_output": {
                "signature": "subprocess.check_output(args, *, shell=False, ...) -> bytes",
                "description": "Runs a command and returns stdout. Sink under shell=True.",
                "role": "sink",
                "tracks": [0],
            },
            "Popen": {
                "signature": "subprocess.Popen(args, *, shell=False, ...) -> Popen",
                "description": "Low-level process constructor. Sink when args is a string with shell=True.",
                "role": "sink",
                "tracks": [0],
            },
        },
        "example_rule": """\
from codepathfinder.python_decorators import python_rule
from codepathfinder import calls, flows
from codepathfinder.presets import PropagationPresets


@python_rule(
    id="PYTHON-CMDI-001",
    name="Command injection via subprocess with shell=True",
    severity="CRITICAL",
    category="command-execution",
    cwe="CWE-78",
    owasp="A03:2021",
    message="User input flows into subprocess with shell=True. Pass args as a list and avoid shell=True.",
)
def detect_subprocess_shell_injection():
    return flows(
        from_sources=[
            calls("request.args.get", "request.form.get", "request.get_json"),
            calls("input"),
        ],
        to_sinks=[
            calls("subprocess.run", match_name={"shell": True}).tracks(0),
            calls("subprocess.Popen", match_name={"shell": True}).tracks(0),
            calls("subprocess.call", match_name={"shell": True}).tracks(0),
            calls("subprocess.check_output", match_name={"shell": True}).tracks(0),
        ],
        sanitized_by=[calls("shlex.quote"), calls("shlex.split")],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
""",
        "rules_using": [],
    },

    "PyOS": {
        "description": (
            "The os standard library module. os.system() and os.popen() always invoke a shell "
            "and are injection sinks. os.exec* variants avoid the shell but are still sinks "
            "for the program path. Environment accessors (os.environ, os.getenv) are sources."
        ),
        "category": "command-execution",
        "fqns": ["os"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "system": {
                "signature": "os.system(command: str) -> int",
                "description": "Executes command via the shell. Command-injection sink.",
                "role": "sink",
                "tracks": [0],
            },
            "popen": {
                "signature": "os.popen(command: str, mode: str = 'r') -> IO",
                "description": "Opens a pipe to a shell command. Injection sink.",
                "role": "sink",
                "tracks": [0],
            },
            "execv": {
                "signature": "os.execv(path: str, args: list) -> None",
                "description": "Replaces the current process. Sink for user-controlled program path.",
                "role": "sink",
                "tracks": [0, 1],
            },
            "execvp": {
                "signature": "os.execvp(file: str, args: list) -> None",
                "description": "Like execv but searches PATH. Same injection risk.",
                "role": "sink",
                "tracks": [0, 1],
            },
            "spawnv": {
                "signature": "os.spawnv(mode: int, path: str, args: list) -> int",
                "description": "Spawns a new process. Sink for user-controlled program path.",
                "role": "sink",
                "tracks": [1, 2],
            },
            "getenv": {
                "signature": "os.getenv(key: str, default: str | None = None) -> str | None",
                "description": "Reads environment variable. Source when attacker controls env (container / CI).",
                "role": "source",
                "tracks": ["return"],
            },
            "environ": {
                "signature": "os.environ: dict[str, str]",
                "description": "Process environment map. Reading from it is a source.",
                "role": "source",
                "tracks": ["return"],
            },
        },
        "rules_using": [],
    },

    # =====================================================================
    # Deserialization
    # =====================================================================

    "PyPickle": {
        "description": (
            "The pickle module for Python object serialization. pickle.load() and pickle.loads() "
            "execute arbitrary code during deserialization via __reduce__ — always unsafe with "
            "untrusted input. Use json or signed payloads instead."
        ),
        "category": "deserialization",
        "fqns": ["pickle"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "load": {
                "signature": "pickle.load(file: IO) -> Any",
                "description": "Reads a pickled object from a file. Arbitrary-code-execution sink on untrusted data.",
                "role": "sink",
                "tracks": [0],
            },
            "loads": {
                "signature": "pickle.loads(data: bytes) -> Any",
                "description": "Deserializes a pickle byte string. RCE sink on untrusted data.",
                "role": "sink",
                "tracks": [0],
            },
            "Unpickler": {
                "signature": "pickle.Unpickler(file: IO) -> Unpickler",
                "description": "Stateful unpickler. The load() method is the sink.",
                "role": "sink",
                "tracks": [0],
            },
        },
        "example_rule": """\
from codepathfinder.python_decorators import python_rule
from codepathfinder import calls, flows
from codepathfinder.presets import PropagationPresets


@python_rule(
    id="PYTHON-DESER-001",
    name="Unsafe Pickle Deserialization",
    severity="CRITICAL",
    category="deserialization",
    cwe="CWE-502",
    owasp="A08:2021",
    message="Untrusted data flows to pickle.loads(). Use json.loads() or signed payloads.",
)
def detect_pickle_deserialization():
    return flows(
        from_sources=[
            calls("request.data"),
            calls("request.get_data"),
            calls("*.read"),
            calls("*.recv"),
        ],
        to_sinks=[calls("pickle.loads"), calls("pickle.load")],
        sanitized_by=[calls("hmac.compare_digest"), calls("*.verify_signature")],
        propagates_through=PropagationPresets.standard(),
        scope="local",
    )
""",
        "rules_using": ["PYTHON-DESER-001"],
    },

    "PyMarshal": {
        "description": (
            "The marshal module for Python internal object serialization. Like pickle, "
            "marshal.load() / marshal.loads() execute code paths determined by the input bytes — "
            "unsafe on untrusted data. The module is undocumented for general use."
        ),
        "category": "deserialization",
        "fqns": ["marshal"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "load": {
                "signature": "marshal.load(file: IO) -> Any",
                "description": "Reads a marshalled object. Unsafe deserialization sink.",
                "role": "sink",
                "tracks": [0],
            },
            "loads": {
                "signature": "marshal.loads(bytes) -> Any",
                "description": "Deserializes marshal bytes. Unsafe on untrusted input.",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": [],
    },

    # =====================================================================
    # Databases — stdlib
    # =====================================================================

    "PySqlite3": {
        "description": (
            "The sqlite3 module wraps the SQLite C library. cursor.execute() and executemany() "
            "accept raw SQL strings and are injection sinks when the SQL is built from user input. "
            "Use the ? placeholder form for safe parameter binding."
        ),
        "category": "databases",
        "fqns": ["sqlite3", "sqlite3.Cursor", "sqlite3.Connection"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "connect": {
                "signature": "sqlite3.connect(database: str, ...) -> Connection",
                "description": "Opens a database connection. Neutral; the Cursor is where injection happens.",
                "role": "neutral",
                "tracks": [],
            },
            "execute": {
                "signature": "Cursor.execute(sql: str, parameters: Sequence = ()) -> Cursor",
                "description": "Executes SQL. Sink for injection when sql is built from user input without placeholders.",
                "role": "sink",
                "tracks": [0],
            },
            "executemany": {
                "signature": "Cursor.executemany(sql: str, parameters: Iterable) -> Cursor",
                "description": "Executes SQL repeatedly. Same injection risk as execute().",
                "role": "sink",
                "tracks": [0],
            },
            "executescript": {
                "signature": "Cursor.executescript(sql_script: str) -> Cursor",
                "description": "Runs a multi-statement SQL script. No parameter binding available — always injection-sensitive.",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": [],
    },

    # =====================================================================
    # Databases — third party
    # =====================================================================

    "PyPsycopg2": {
        "description": (
            "psycopg2 is the canonical PostgreSQL driver for Python. Cursor.execute() and "
            "executemany() are SQL injection sinks when the query is built by string concatenation "
            "or f-strings. Use %s placeholders for safe binding."
        ),
        "category": "databases",
        "fqns": ["psycopg2", "psycopg2.extensions.cursor"],
        "pip_snippet": "pip install psycopg2-binary",
        "methods": {
            "connect": {
                "signature": "psycopg2.connect(dsn=None, ...) -> Connection",
                "description": "Opens a PostgreSQL connection.",
                "role": "neutral",
                "tracks": [],
            },
            "execute": {
                "signature": "cursor.execute(query: str, vars: Sequence | Mapping = None) -> None",
                "description": "Executes a query. SQL injection sink when query is built from user input.",
                "role": "sink",
                "tracks": [0],
            },
            "executemany": {
                "signature": "cursor.executemany(query: str, vars_list: Iterable) -> None",
                "description": "Executes a query for each element in vars_list. Same injection risk.",
                "role": "sink",
                "tracks": [0],
            },
            "mogrify": {
                "signature": "cursor.mogrify(query: str, vars=None) -> bytes",
                "description": "Returns the query after parameter substitution. Does not execute, but the resulting bytes can flow to a later execute().",
                "role": "neutral",
                "tracks": [],
            },
        },
        "rules_using": [],
    },

    "PyPyMongo": {
        "description": (
            "PyMongo is the official MongoDB driver for Python. Collection methods accept filter "
            "dicts; NoSQL injection occurs when filter dicts are built from user-supplied JSON "
            "that lets attackers inject $where, $regex, or operator keys."
        ),
        "category": "databases",
        "fqns": ["pymongo", "pymongo.collection.Collection", "pymongo.MongoClient"],
        "pip_snippet": "pip install pymongo",
        "methods": {
            "find": {
                "signature": "Collection.find(filter: Mapping = None, projection: Mapping = None, ...) -> Cursor",
                "description": "Queries documents. NoSQL injection sink if filter is built from user input.",
                "role": "sink",
                "tracks": [0],
            },
            "find_one": {
                "signature": "Collection.find_one(filter: Mapping = None, ...) -> dict | None",
                "description": "Returns first matching document. Same NoSQL injection risk.",
                "role": "sink",
                "tracks": [0],
            },
            "update_one": {
                "signature": "Collection.update_one(filter: Mapping, update: Mapping, ...) -> UpdateResult",
                "description": "Updates a single document. Injection sink on filter and update args.",
                "role": "sink",
                "tracks": [0, 1],
            },
            "delete_one": {
                "signature": "Collection.delete_one(filter: Mapping, ...) -> DeleteResult",
                "description": "Deletes a single document. NoSQL injection sink.",
                "role": "sink",
                "tracks": [0],
            },
            "aggregate": {
                "signature": "Collection.aggregate(pipeline: Sequence[Mapping], ...) -> CommandCursor",
                "description": "Runs an aggregation pipeline. Each stage can be injection-sensitive.",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": [],
    },

    "PyRedis": {
        "description": (
            "redis-py is the de-facto Redis client for Python. Most commands are typed and safe. "
            "The main sinks are eval() and evalsha() which run Lua scripts — injection-sensitive "
            "when the script body is user-controlled."
        ),
        "category": "databases",
        "fqns": ["redis", "redis.Redis", "redis.StrictRedis"],
        "pip_snippet": "pip install redis",
        "methods": {
            "eval": {
                "signature": "Redis.eval(script: str, numkeys: int, *keys_and_args) -> Any",
                "description": "Executes a Lua script on the server. Injection sink when script is user-controlled.",
                "role": "sink",
                "tracks": [0],
            },
            "evalsha": {
                "signature": "Redis.evalsha(sha: str, numkeys: int, *keys_and_args) -> Any",
                "description": "Executes a cached Lua script by SHA. Tainted sha reaches pre-registered scripts.",
                "role": "sink",
                "tracks": [0],
            },
            "execute_command": {
                "signature": "Redis.execute_command(*args) -> Any",
                "description": "Sends an arbitrary Redis command. Injection sink for command name.",
                "role": "sink",
                "tracks": [0],
            },
            "get": {
                "signature": "Redis.get(name: str) -> bytes | None",
                "description": "Reads a key. Source when cached data originated from user input.",
                "role": "source",
                "tracks": ["return"],
            },
            "set": {
                "signature": "Redis.set(name: str, value, ex=None, ...) -> bool",
                "description": "Sets a key. Typed arguments, generally safe.",
                "role": "neutral",
                "tracks": [],
            },
        },
        "rules_using": [],
    },

    # =====================================================================
    # Web frameworks
    # =====================================================================

    "PyFlask": {
        "description": (
            "Flask is a popular Python web microframework. The flask.request global exposes all "
            "HTTP input (args, form, json, files, headers, cookies) as taint sources. Response helpers "
            "like render_template (SSTI if template is user-controlled) and redirect (open-redirect) are sinks."
        ),
        "category": "web-frameworks",
        "fqns": ["flask", "flask.Request"],
        "pip_snippet": "pip install flask",
        "methods": {
            "request.args": {
                "signature": "request.args: MultiDict",
                "description": "URL query string. All values are user-controlled.",
                "role": "source",
                "tracks": ["return"],
            },
            "request.form": {
                "signature": "request.form: MultiDict",
                "description": "POST form data (application/x-www-form-urlencoded, multipart/form-data).",
                "role": "source",
                "tracks": ["return"],
            },
            "request.get_json": {
                "signature": "request.get_json(force=False, silent=False, cache=True) -> Any",
                "description": "Parsed JSON request body. User-controlled.",
                "role": "source",
                "tracks": ["return"],
            },
            "request.cookies": {
                "signature": "request.cookies: ImmutableMultiDict",
                "description": "Request cookies. User-controlled.",
                "role": "source",
                "tracks": ["return"],
            },
            "request.headers": {
                "signature": "request.headers: EnvironHeaders",
                "description": "Request headers. User-controlled.",
                "role": "source",
                "tracks": ["return"],
            },
            "render_template_string": {
                "signature": "flask.render_template_string(source: str, **context) -> str",
                "description": "Renders a template from a raw string. SSTI sink when source contains user input.",
                "role": "sink",
                "tracks": [0],
            },
            "redirect": {
                "signature": "flask.redirect(location: str, code: int = 302) -> Response",
                "description": "Returns a redirect response. Open-redirect sink when location is user-controlled.",
                "role": "sink",
                "tracks": [0],
            },
            "send_file": {
                "signature": "flask.send_file(path_or_file, ...) -> Response",
                "description": "Serves a file. Path-traversal sink when path is user-controlled.",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": [],
    },

    "PyDjango": {
        "description": (
            "Django is a full-featured Python web framework. HttpRequest exposes request data; "
            "the ORM Manager.raw() and Cursor.execute() are SQL injection sinks when the SQL is built "
            "from user input. Template rendering via mark_safe bypasses auto-escaping (XSS sink)."
        ),
        "category": "web-frameworks",
        "fqns": ["django", "django.http.HttpRequest", "django.db.models.Manager"],
        "pip_snippet": "pip install django",
        "methods": {
            "HttpRequest.GET": {
                "signature": "request.GET: QueryDict",
                "description": "URL query parameters. All values user-controlled.",
                "role": "source",
                "tracks": ["return"],
            },
            "HttpRequest.POST": {
                "signature": "request.POST: QueryDict",
                "description": "POST form data. User-controlled.",
                "role": "source",
                "tracks": ["return"],
            },
            "HttpRequest.COOKIES": {
                "signature": "request.COOKIES: dict[str, str]",
                "description": "Request cookies. User-controlled.",
                "role": "source",
                "tracks": ["return"],
            },
            "HttpRequest.body": {
                "signature": "request.body: bytes",
                "description": "Raw HTTP request body. User-controlled.",
                "role": "source",
                "tracks": ["return"],
            },
            "Manager.raw": {
                "signature": "Manager.raw(raw_query: str, params: Sequence = None, ...) -> RawQuerySet",
                "description": "Executes a raw SQL query against the ORM. SQL injection sink when raw_query is built from user input.",
                "role": "sink",
                "tracks": [0],
            },
            "mark_safe": {
                "signature": "django.utils.safestring.mark_safe(s: str) -> SafeString",
                "description": "Declares a string as safe, bypassing template auto-escaping. XSS sink on user input.",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": [],
    },

    "PyFastAPI": {
        "description": (
            "FastAPI is a modern Python web framework built on Starlette and Pydantic. Path / query / "
            "body parameters declared on endpoints are sources. Response helpers inherited from "
            "Starlette include HTMLResponse and RedirectResponse (XSS and open-redirect sinks)."
        ),
        "category": "web-frameworks",
        "fqns": ["fastapi", "fastapi.Request", "starlette.requests.Request"],
        "pip_snippet": "pip install fastapi",
        "methods": {
            "Request.query_params": {
                "signature": "request.query_params: QueryParams",
                "description": "URL query parameters. User-controlled.",
                "role": "source",
                "tracks": ["return"],
            },
            "Request.cookies": {
                "signature": "request.cookies: dict[str, str]",
                "description": "Request cookies. User-controlled.",
                "role": "source",
                "tracks": ["return"],
            },
            "Request.headers": {
                "signature": "request.headers: Headers",
                "description": "Request headers. User-controlled.",
                "role": "source",
                "tracks": ["return"],
            },
            "Request.json": {
                "signature": "async Request.json() -> Any",
                "description": "Parsed JSON request body. User-controlled.",
                "role": "source",
                "tracks": ["return"],
            },
            "HTMLResponse": {
                "signature": "HTMLResponse(content: str, status_code: int = 200, ...) -> Response",
                "description": "Returns raw HTML. XSS sink when content contains unescaped user input.",
                "role": "sink",
                "tracks": [0],
            },
            "RedirectResponse": {
                "signature": "RedirectResponse(url: str, status_code: int = 307, ...) -> Response",
                "description": "Returns a redirect. Open-redirect sink when url is user-controlled.",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": [],
    },

    # =====================================================================
    # Templating
    # =====================================================================

    "PyJinja2": {
        "description": (
            "Jinja2 is the template engine behind Flask and many Python frameworks. "
            "Template(source).render() and Environment.from_string() evaluate template syntax — "
            "SSTI sink when the template source comes from user input. Autoescape only protects "
            "rendered output, not the template source itself."
        ),
        "category": "templating",
        "fqns": ["jinja2", "jinja2.Template", "jinja2.Environment"],
        "pip_snippet": "pip install jinja2",
        "methods": {
            "Template": {
                "signature": "jinja2.Template(source: str, ...) -> Template",
                "description": "Compiles a template from source. SSTI sink when source is user-controlled.",
                "role": "sink",
                "tracks": [0],
            },
            "Environment.from_string": {
                "signature": "Environment.from_string(source: str, ...) -> Template",
                "description": "Compiles a template from source using this environment. Same SSTI risk.",
                "role": "sink",
                "tracks": [0],
            },
            "render": {
                "signature": "Template.render(**context) -> str",
                "description": "Renders a compiled template. Safe with autoescape=True on trusted templates; dangerous if the Template source itself was user-controlled.",
                "role": "neutral",
                "tracks": [],
            },
        },
        "rules_using": [],
    },

    # =====================================================================
    # HTTP clients
    # =====================================================================

    "PyRequests": {
        "description": (
            "requests is the most popular HTTP client for Python. All top-level methods and "
            "Session methods accept a URL as the first argument — SSRF sink when the URL is "
            "user-controlled. verify=False disables TLS verification (separate rule)."
        ),
        "category": "http-clients",
        "fqns": ["requests", "requests.Session"],
        "pip_snippet": "pip install requests",
        "methods": {
            "get": {
                "signature": "requests.get(url: str, params=None, **kwargs) -> Response",
                "description": "Sends a GET request. SSRF sink when url is user-controlled.",
                "role": "sink",
                "tracks": [0],
            },
            "post": {
                "signature": "requests.post(url: str, data=None, json=None, **kwargs) -> Response",
                "description": "Sends a POST request. SSRF sink when url is user-controlled.",
                "role": "sink",
                "tracks": [0],
            },
            "put": {
                "signature": "requests.put(url: str, data=None, **kwargs) -> Response",
                "description": "Sends a PUT request. SSRF sink.",
                "role": "sink",
                "tracks": [0],
            },
            "delete": {
                "signature": "requests.delete(url: str, **kwargs) -> Response",
                "description": "Sends a DELETE request. SSRF sink.",
                "role": "sink",
                "tracks": [0],
            },
            "request": {
                "signature": "requests.request(method: str, url: str, **kwargs) -> Response",
                "description": "Sends a request with arbitrary method. SSRF sink on url.",
                "role": "sink",
                "tracks": [1],
            },
        },
        "rules_using": [],
    },

    "PyUrllib": {
        "description": (
            "urllib.request (stdlib) is the lowest-level HTTP client in Python. urlopen() accepts "
            "both a URL string and a Request object — SSRF sink when the URL is user-controlled. "
            "Unlike requests, urlopen defaults to no TLS verification on some platforms."
        ),
        "category": "http-clients",
        "fqns": ["urllib.request"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "urlopen": {
                "signature": "urllib.request.urlopen(url, data=None, timeout=None, ...) -> HTTPResponse",
                "description": "Opens an HTTP(S) URL. SSRF sink when url is user-controlled.",
                "role": "sink",
                "tracks": [0],
            },
            "Request": {
                "signature": "urllib.request.Request(url: str, data=None, headers={}, ...) -> Request",
                "description": "Builds an HTTP request object. SSRF sink when url is user-controlled (passed later to urlopen).",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": [],
    },

    # =====================================================================
    # Deserialization — third party
    # =====================================================================

    "PyYaml": {
        "description": (
            "PyYAML is the standard YAML library. yaml.load() with the default Loader (or "
            "UnsafeLoader / Loader) instantiates arbitrary Python classes — RCE sink on untrusted "
            "input. Use yaml.safe_load() or Loader=yaml.SafeLoader instead."
        ),
        "category": "deserialization",
        "fqns": ["yaml"],
        "pip_snippet": "pip install pyyaml",
        "methods": {
            "load": {
                "signature": "yaml.load(stream, Loader) -> Any",
                "description": "Deserializes a YAML document. RCE sink under Loader / UnsafeLoader.",
                "role": "sink",
                "tracks": [0],
            },
            "load_all": {
                "signature": "yaml.load_all(stream, Loader) -> Iterator[Any]",
                "description": "Deserializes multiple YAML documents. Same RCE risk as load().",
                "role": "sink",
                "tracks": [0],
            },
            "full_load": {
                "signature": "yaml.full_load(stream) -> Any",
                "description": "Uses FullLoader — safer than Loader but still resolves some Python tags. Prefer safe_load.",
                "role": "sink",
                "tracks": [0],
            },
            "safe_load": {
                "signature": "yaml.safe_load(stream) -> Any",
                "description": "Deserializes YAML using SafeLoader. Only built-in types. Use this.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
            "safe_load_all": {
                "signature": "yaml.safe_load_all(stream) -> Iterator[Any]",
                "description": "Safe multi-document load. Sanitizer.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
        },
        "rules_using": [],
    },

    # =====================================================================
    # File system
    # =====================================================================

    "PyOSPath": {
        "description": (
            "The os.path module for path manipulation. join() concatenates path components but "
            "does not resolve traversal sequences — path-traversal bug when joining a trusted "
            "base with a user-controlled path. Use os.path.commonpath + realpath containment "
            "checks to sanitize."
        ),
        "category": "file-system",
        "fqns": ["os.path"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "join": {
                "signature": "os.path.join(*paths: str) -> str",
                "description": "Joins path components. Does NOT defend against ../ traversal — neutral, but the output often reaches file sinks.",
                "role": "neutral",
                "tracks": [],
            },
            "abspath": {
                "signature": "os.path.abspath(path: str) -> str",
                "description": "Returns the absolute path. Does not resolve symlinks.",
                "role": "neutral",
                "tracks": [],
            },
            "realpath": {
                "signature": "os.path.realpath(path: str) -> str",
                "description": "Resolves all symlinks and . / .. components. Combine with commonpath for traversal defense.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
            "commonpath": {
                "signature": "os.path.commonpath(paths: Sequence[str]) -> str",
                "description": "Returns the longest common path. Use to assert a user path stays inside a trusted base.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
        },
        "rules_using": [],
    },

    "PyTempfile": {
        "description": (
            "The tempfile module. mktemp() is deprecated and insecure (race condition between "
            "filename generation and open). Use NamedTemporaryFile, mkstemp, or TemporaryDirectory "
            "which atomically create the file."
        ),
        "category": "file-system",
        "fqns": ["tempfile"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "mktemp": {
                "signature": "tempfile.mktemp(suffix='', prefix='tmp', dir=None) -> str",
                "description": "Returns a candidate temp file path without creating it. Insecure (TOCTOU) — finding whenever used.",
                "role": "sink",
                "tracks": [],
            },
            "mkstemp": {
                "signature": "tempfile.mkstemp(suffix=None, prefix=None, dir=None, text=False) -> (fd, path)",
                "description": "Atomically creates a temp file and returns an open fd. Safe replacement for mktemp.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
            "NamedTemporaryFile": {
                "signature": "tempfile.NamedTemporaryFile(mode='w+b', ...) -> _TemporaryFileWrapper",
                "description": "Context-managed temp file. Atomic creation. Safe.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
        },
        "rules_using": [],
    },

    # =====================================================================
    # Archives
    # =====================================================================

    "PyTarfile": {
        "description": (
            "The tarfile module. extractall() and extract() follow archive entry paths as-is — "
            "path-traversal sink (zip slip) when the archive is user-supplied and extractall's "
            "filter= argument is not set to a safe filter. Python 3.12 changed the default to 'data'."
        ),
        "category": "archives",
        "fqns": ["tarfile", "tarfile.TarFile"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "extractall": {
                "signature": "TarFile.extractall(path='.', members=None, *, numeric_owner=False, filter=None) -> None",
                "description": "Extracts all entries. Zip-slip sink when members comes from a hostile archive and filter is unset.",
                "role": "sink",
                "tracks": [],
            },
            "extract": {
                "signature": "TarFile.extract(member, path='', *, set_attrs=True, numeric_owner=False, filter=None) -> None",
                "description": "Extracts a single entry. Same path-traversal risk as extractall.",
                "role": "sink",
                "tracks": [],
            },
            "open": {
                "signature": "tarfile.open(name=None, mode='r', fileobj=None, ...) -> TarFile",
                "description": "Opens a tar archive. Neutral; extract() is where traversal happens.",
                "role": "neutral",
                "tracks": [],
            },
        },
        "rules_using": [],
    },

    "PyZipfile": {
        "description": (
            "The zipfile module. ZipFile.extractall() and extract() are zip-slip sinks when the "
            "archive is untrusted. Python's extractall resolves .. segments in archive members "
            "to paths outside the target directory."
        ),
        "category": "archives",
        "fqns": ["zipfile", "zipfile.ZipFile"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "extractall": {
                "signature": "ZipFile.extractall(path=None, members=None, pwd=None) -> None",
                "description": "Extracts all members. Zip-slip sink on untrusted archives.",
                "role": "sink",
                "tracks": [],
            },
            "extract": {
                "signature": "ZipFile.extract(member, path=None, pwd=None) -> str",
                "description": "Extracts a single member. Same zip-slip risk.",
                "role": "sink",
                "tracks": [],
            },
        },
        "rules_using": [],
    },

    # =====================================================================
    # Cryptography
    # =====================================================================

    "PyHashlib": {
        "description": (
            "The hashlib module provides cryptographic hash functions. md5 and sha1 are "
            "cryptographically broken — findings for password hashing / signature use. For "
            "password hashing use hashlib.scrypt, pbkdf2_hmac, or the passlib / argon2-cffi packages."
        ),
        "category": "crypto",
        "fqns": ["hashlib"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "md5": {
                "signature": "hashlib.md5(data: bytes = b'', *, usedforsecurity=True) -> Hash",
                "description": "MD5 hash. Broken for cryptographic use — finding for password hashing or digital signatures.",
                "role": "sink",
                "tracks": [],
            },
            "sha1": {
                "signature": "hashlib.sha1(data: bytes = b'', *, usedforsecurity=True) -> Hash",
                "description": "SHA-1 hash. Broken for cryptographic use — finding for signature contexts.",
                "role": "sink",
                "tracks": [],
            },
            "sha256": {
                "signature": "hashlib.sha256(data: bytes = b'') -> Hash",
                "description": "SHA-256 hash. Acceptable for digests; use scrypt / pbkdf2 for passwords.",
                "role": "neutral",
                "tracks": [],
            },
            "pbkdf2_hmac": {
                "signature": "hashlib.pbkdf2_hmac(hash_name, password, salt, iterations, dklen=None) -> bytes",
                "description": "Password-based key derivation. Safe with iterations ≥ 100_000.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
            "scrypt": {
                "signature": "hashlib.scrypt(password, *, salt, n, r, p, maxmem=0, dklen=64) -> bytes",
                "description": "Memory-hard password hash. Safe.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
        },
        "rules_using": [],
    },

    "PyHmac": {
        "description": (
            "The hmac module for keyed message authentication. compare_digest is the only "
            "constant-time comparison helper — using ordinary == for MAC comparison is a timing-attack sink."
        ),
        "category": "crypto",
        "fqns": ["hmac"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "new": {
                "signature": "hmac.new(key: bytes, msg: bytes = None, digestmod='') -> HMAC",
                "description": "Creates an HMAC instance. Neutral.",
                "role": "neutral",
                "tracks": [],
            },
            "compare_digest": {
                "signature": "hmac.compare_digest(a, b) -> bool",
                "description": "Constant-time comparison. Sanitizer for signature verification flows.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
        },
        "rules_using": [],
    },

    "PySecrets": {
        "description": (
            "The secrets module provides cryptographically strong random values suitable for "
            "managing authentication tokens. Use secrets instead of the random module for "
            "session IDs, tokens, and CSRF nonces."
        ),
        "category": "crypto",
        "fqns": ["secrets"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "token_bytes": {
                "signature": "secrets.token_bytes(nbytes: int | None = None) -> bytes",
                "description": "Cryptographically secure random bytes. Safe source for tokens.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
            "token_hex": {
                "signature": "secrets.token_hex(nbytes: int | None = None) -> str",
                "description": "Hex-encoded secure random token. Safe.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
            "token_urlsafe": {
                "signature": "secrets.token_urlsafe(nbytes: int | None = None) -> str",
                "description": "URL-safe base64 secure random token. Safe.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
            "compare_digest": {
                "signature": "secrets.compare_digest(a, b) -> bool",
                "description": "Constant-time comparison. Sanitizer for secret comparison.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
            "choice": {
                "signature": "secrets.choice(seq)",
                "description": "Cryptographically secure random choice from a non-empty sequence.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
        },
        "rules_using": [],
    },

    "PyRandom": {
        "description": (
            "The random module uses a Mersenne Twister PRNG — NOT suitable for cryptography. "
            "random.random, random.choice, random.randint, and SystemRandom(..) should be flagged "
            "for security contexts. Use the secrets module for tokens, passwords, and keys."
        ),
        "category": "crypto",
        "fqns": ["random"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "random": {
                "signature": "random.random() -> float",
                "description": "Non-crypto float in [0.0, 1.0). Sink for security-relevant randomness.",
                "role": "sink",
                "tracks": [],
            },
            "randint": {
                "signature": "random.randint(a: int, b: int) -> int",
                "description": "Non-crypto integer in [a, b]. Sink for security-relevant randomness.",
                "role": "sink",
                "tracks": [],
            },
            "choice": {
                "signature": "random.choice(seq)",
                "description": "Non-crypto random choice. Sink for tokens / passwords / keys.",
                "role": "sink",
                "tracks": [],
            },
            "randbytes": {
                "signature": "random.randbytes(n: int) -> bytes",
                "description": "Non-crypto random bytes. Sink for cryptographic use.",
                "role": "sink",
                "tracks": [],
            },
            "seed": {
                "signature": "random.seed(a=None, version=2) -> None",
                "description": "Seeds the PRNG. Findings when seeded with predictable value for security-sensitive randomness.",
                "role": "sink",
                "tracks": [],
            },
        },
        "rules_using": [],
    },

    "PySsl": {
        "description": (
            "The ssl module for TLS / SSL. SSLContext with verify_mode=CERT_NONE disables "
            "certificate validation (MITM risk). _create_unverified_context() is an explicit "
            "bypass — finding for any production code. Use create_default_context() for sane defaults."
        ),
        "category": "crypto",
        "fqns": ["ssl"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "create_default_context": {
                "signature": "ssl.create_default_context(purpose=Purpose.SERVER_AUTH, ...) -> SSLContext",
                "description": "Creates a context with safe defaults (verify, hostname check). Sanitizer.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
            "_create_unverified_context": {
                "signature": "ssl._create_unverified_context() -> SSLContext",
                "description": "Returns a context that skips verification. Always a finding in production code.",
                "role": "sink",
                "tracks": [],
            },
            "wrap_socket": {
                "signature": "ssl.wrap_socket(sock, ssl_version=..., cert_reqs=CERT_NONE, ...) -> SSLSocket",
                "description": "Legacy socket wrapping. Finding when cert_reqs=CERT_NONE.",
                "role": "sink",
                "tracks": [],
            },
            "SSLContext": {
                "signature": "ssl.SSLContext(protocol=PROTOCOL_TLS) -> SSLContext",
                "description": "TLS context. Finding when .check_hostname is False or .verify_mode is CERT_NONE.",
                "role": "neutral",
                "tracks": [],
            },
        },
        "rules_using": [],
    },

    # =====================================================================
    # Stdlib — parsers & eval
    # =====================================================================

    "PyAst": {
        "description": (
            "The ast module exposes Python's abstract syntax tree. ast.literal_eval is a safe "
            "evaluator for literals only. The builtins eval() and exec() execute arbitrary Python "
            "code — RCE sinks on user input. compile() produces code objects that reach exec()."
        ),
        "category": "deserialization",
        "fqns": ["ast", "builtins"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "literal_eval": {
                "signature": "ast.literal_eval(node_or_string) -> Any",
                "description": "Safely evaluates Python literals (str, int, list, dict, tuple, bool, None). Sanitizer replacement for eval().",
                "role": "sanitizer",
                "tracks": ["return"],
            },
            "parse": {
                "signature": "ast.parse(source, filename='<unknown>', mode='exec', ...) -> Module",
                "description": "Parses source into an AST. Neutral on its own.",
                "role": "neutral",
                "tracks": [],
            },
            "eval": {
                "signature": "eval(expression, globals=None, locals=None) -> Any",
                "description": "Evaluates a Python expression. RCE sink when expression is user-controlled.",
                "role": "sink",
                "tracks": [0],
            },
            "exec": {
                "signature": "exec(object, globals=None, locals=None) -> None",
                "description": "Executes Python code. RCE sink on user-controlled source.",
                "role": "sink",
                "tracks": [0],
            },
            "compile": {
                "signature": "compile(source, filename, mode, flags=0, dont_inherit=False, optimize=-1)",
                "description": "Compiles source to a code object. Reaches exec / eval. Sink on user-controlled source.",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": [],
    },

    "PyJson": {
        "description": (
            "The json module for JSON encode / decode. Unlike pickle, json is safe by default — "
            "only parses primitives, lists, dicts. Still worth documenting because json.loads is a "
            "common source entry point and json.dumps on response values is where reflected XSS originates."
        ),
        "category": "deserialization",
        "fqns": ["json"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "loads": {
                "signature": "json.loads(s: str | bytes, ...) -> Any",
                "description": "Parses a JSON string. Safe by default. Source for user-controlled JSON input.",
                "role": "source",
                "tracks": ["return"],
            },
            "load": {
                "signature": "json.load(fp, ...) -> Any",
                "description": "Parses JSON from a file. Safe. Source when fp is user-controlled.",
                "role": "source",
                "tracks": ["return"],
            },
            "dumps": {
                "signature": "json.dumps(obj, *, ensure_ascii=True, ...) -> str",
                "description": "Serializes obj to JSON. Neutral.",
                "role": "neutral",
                "tracks": [],
            },
            "dump": {
                "signature": "json.dump(obj, fp, ...) -> None",
                "description": "Writes JSON to a file. Neutral.",
                "role": "neutral",
                "tracks": [],
            },
        },
        "rules_using": [],
    },

    # =====================================================================
    # Stdlib — HTTP / networking
    # =====================================================================

    "PyHttpClient": {
        "description": (
            "The http.client module provides low-level HTTP primitives. HTTPConnection / "
            "HTTPSConnection.request() is an SSRF sink when the host or path comes from user input. "
            "HTTPSConnection with context=None falls back to system default TLS settings."
        ),
        "category": "http-clients",
        "fqns": ["http.client", "http.client.HTTPConnection", "http.client.HTTPSConnection"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "HTTPConnection": {
                "signature": "http.client.HTTPConnection(host, port=None, ...) -> HTTPConnection",
                "description": "Opens an HTTP connection. SSRF sink when host is user-controlled.",
                "role": "sink",
                "tracks": [0],
            },
            "HTTPSConnection": {
                "signature": "http.client.HTTPSConnection(host, port=None, *, context=None, ...) -> HTTPSConnection",
                "description": "Opens an HTTPS connection. SSRF sink on host. context=None uses defaults.",
                "role": "sink",
                "tracks": [0],
            },
            "request": {
                "signature": "HTTPConnection.request(method: str, url: str, body=None, headers={}) -> None",
                "description": "Sends an HTTP request. SSRF sink when url is user-controlled.",
                "role": "sink",
                "tracks": [1],
            },
        },
        "rules_using": [],
    },

    "PySocket": {
        "description": (
            "The socket module for low-level network operations. socket.connect() is an SSRF "
            "primitive when the host / port comes from user input. socket.bind() on 0.0.0.0 is "
            "a finding for services that should be localhost-only."
        ),
        "category": "http-clients",
        "fqns": ["socket"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "socket": {
                "signature": "socket.socket(family=AF_INET, type=SOCK_STREAM, proto=0, fileno=None) -> socket",
                "description": "Creates a socket. Neutral.",
                "role": "neutral",
                "tracks": [],
            },
            "connect": {
                "signature": "socket.connect(address: tuple | str) -> None",
                "description": "Connects to a remote address. SSRF sink when address is user-controlled.",
                "role": "sink",
                "tracks": [0],
            },
            "bind": {
                "signature": "socket.bind(address: tuple | str) -> None",
                "description": "Binds to a local address. Finding when bound to 0.0.0.0 or '' on internal services.",
                "role": "sink",
                "tracks": [0],
            },
            "create_connection": {
                "signature": "socket.create_connection(address, timeout=..., source_address=None) -> socket",
                "description": "High-level connection helper. SSRF sink on address.",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": [],
    },

    "PyUrllibParse": {
        "description": (
            "The urllib.parse module for URL parsing and building. urljoin is commonly used to "
            "build request URLs — when the base is user-controlled, attackers can redirect to "
            "arbitrary hosts. urlparse can be used as a sanitizer for SSRF if the netloc is validated."
        ),
        "category": "http-clients",
        "fqns": ["urllib.parse"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "urlparse": {
                "signature": "urllib.parse.urlparse(urlstring: str, scheme='', allow_fragments=True) -> ParseResult",
                "description": "Parses a URL into components. Building block for SSRF sanitization (check netloc).",
                "role": "neutral",
                "tracks": ["return"],
            },
            "urljoin": {
                "signature": "urllib.parse.urljoin(base: str, url: str, allow_fragments=True) -> str",
                "description": "Joins a base URL and a relative URL. Neutral; output often reaches HTTP sinks.",
                "role": "neutral",
                "tracks": ["return"],
            },
            "quote": {
                "signature": "urllib.parse.quote(string, safe='/', ...) -> str",
                "description": "Percent-encodes a URL component. Sanitizer when used on user input before URL concat.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
            "quote_plus": {
                "signature": "urllib.parse.quote_plus(string, safe='', ...) -> str",
                "description": "Like quote but encodes spaces as +. Sanitizer for query strings.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
        },
        "rules_using": [],
    },

    "PyFtplib": {
        "description": (
            "The ftplib module for FTP (insecure plaintext protocol). FTP() connects unencrypted; "
            "FTP_TLS is the secure variant. Any use of the plain FTP class is a finding for sensitive "
            "data flows."
        ),
        "category": "http-clients",
        "fqns": ["ftplib"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "FTP": {
                "signature": "ftplib.FTP(host='', user='', passwd='', acct='', timeout=...) -> FTP",
                "description": "Opens a plaintext FTP session. Finding — credentials transmitted unencrypted.",
                "role": "sink",
                "tracks": [],
            },
            "FTP_TLS": {
                "signature": "ftplib.FTP_TLS(host='', user='', passwd='', ...) -> FTP_TLS",
                "description": "Opens an FTPS session. Secure replacement.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
        },
        "rules_using": [],
    },

    "PyTelnetlib": {
        "description": (
            "The telnetlib module for Telnet (insecure plaintext protocol). Any use of Telnet is "
            "a finding; use paramiko / SSH instead. Deprecated since 3.11, removed in 3.13."
        ),
        "category": "http-clients",
        "fqns": ["telnetlib"],
        "pip_snippet": "# stdlib — no install required (removed in Python 3.13)",
        "methods": {
            "Telnet": {
                "signature": "telnetlib.Telnet(host=None, port=0, timeout=...) -> Telnet",
                "description": "Opens a plaintext Telnet session. Finding — deprecated and insecure.",
                "role": "sink",
                "tracks": [],
            },
        },
        "rules_using": [],
    },

    "PySmtplib": {
        "description": (
            "The smtplib module for SMTP. SMTP() uses plaintext unless starttls() is called. "
            "SMTP_SSL is the always-encrypted variant. Rule writers also target email header / "
            "recipient construction for header-injection sinks."
        ),
        "category": "http-clients",
        "fqns": ["smtplib"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "SMTP": {
                "signature": "smtplib.SMTP(host='', port=0, local_hostname=None, ...) -> SMTP",
                "description": "Opens a plaintext SMTP session. Finding if starttls is not called later.",
                "role": "sink",
                "tracks": [],
            },
            "SMTP_SSL": {
                "signature": "smtplib.SMTP_SSL(host='', port=0, ..., context=None) -> SMTP_SSL",
                "description": "Opens an SMTP session over TLS. Safe.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
            "sendmail": {
                "signature": "SMTP.sendmail(from_addr, to_addrs, msg, mail_options=(), rcpt_options=())",
                "description": "Sends an email. Header-injection sink when msg / to_addrs is user-controlled without sanitization.",
                "role": "sink",
                "tracks": [1, 2],
            },
        },
        "rules_using": [],
    },

    # =====================================================================
    # Stdlib — file system
    # =====================================================================

    "PyPathlib": {
        "description": (
            "The pathlib module is the modern OO path API. Path.resolve() expands symlinks "
            "(sanitizer when combined with containment check). Path.open / read_text / write_text "
            "are file I/O sinks when the path is user-controlled."
        ),
        "category": "file-system",
        "fqns": ["pathlib", "pathlib.Path"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "Path": {
                "signature": "pathlib.Path(*pathsegments) -> Path",
                "description": "Constructs a path. Neutral; does not defend against traversal.",
                "role": "neutral",
                "tracks": [],
            },
            "resolve": {
                "signature": "Path.resolve(strict=False) -> Path",
                "description": "Resolves symlinks and relative segments. Use with relative_to() for traversal defense.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
            "open": {
                "signature": "Path.open(mode='r', buffering=-1, ...) -> IO",
                "description": "Opens the file at this path. Path-traversal sink when path is user-controlled.",
                "role": "sink",
                "tracks": [],
            },
            "read_text": {
                "signature": "Path.read_text(encoding=None, errors=None) -> str",
                "description": "Reads the whole file as text. Path-traversal sink.",
                "role": "sink",
                "tracks": [],
            },
            "write_text": {
                "signature": "Path.write_text(data, encoding=None, errors=None, newline=None) -> int",
                "description": "Writes text. Path-traversal sink when path is user-controlled.",
                "role": "sink",
                "tracks": [],
            },
        },
        "rules_using": [],
    },

    "PyShutil": {
        "description": (
            "The shutil module for high-level file operations. unpack_archive automatically "
            "extracts tar / zip / gztar / bztar / xztar archives — same zip-slip risks as "
            "tarfile.extractall. copytree can also be used for path-traversal."
        ),
        "category": "file-system",
        "fqns": ["shutil"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "unpack_archive": {
                "signature": "shutil.unpack_archive(filename, extract_dir=None, format=None) -> None",
                "description": "Unpacks an archive. Zip-slip sink on untrusted archives — uses tarfile / zipfile under the hood.",
                "role": "sink",
                "tracks": [],
            },
            "copyfile": {
                "signature": "shutil.copyfile(src, dst, *, follow_symlinks=True) -> str",
                "description": "Copies a file. Path-traversal sink when src / dst is user-controlled.",
                "role": "sink",
                "tracks": [0, 1],
            },
            "copytree": {
                "signature": "shutil.copytree(src, dst, symlinks=False, ...) -> str",
                "description": "Recursively copies a directory. Path-traversal sink on untrusted paths.",
                "role": "sink",
                "tracks": [0, 1],
            },
            "rmtree": {
                "signature": "shutil.rmtree(path, ignore_errors=False, onerror=None) -> None",
                "description": "Recursively deletes a directory tree. Finding on user-controlled path (arbitrary-file-delete).",
                "role": "sink",
                "tracks": [0],
            },
            "which": {
                "signature": "shutil.which(cmd, mode=os.F_OK | os.X_OK, path=None) -> str | None",
                "description": "Locates an executable on PATH. Neutral.",
                "role": "neutral",
                "tracks": [],
            },
        },
        "rules_using": [],
    },

    "PyShlex": {
        "description": (
            "The shlex module provides shell-compatible tokenization and quoting. shlex.quote "
            "is the canonical sanitizer for shell=True command construction. shlex.split is safer "
            "than splitting yourself, but quote is what protects against shell-metacharacter injection."
        ),
        "category": "command-execution",
        "fqns": ["shlex"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "quote": {
                "signature": "shlex.quote(s: str) -> str",
                "description": "Returns a shell-escaped version of s. Sanitizer for shell=True sinks.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
            "split": {
                "signature": "shlex.split(s, comments=False, posix=True) -> list[str]",
                "description": "Splits a string using shell-like syntax. Sanitizer when producing list[str] for subprocess (implies shell=False).",
                "role": "sanitizer",
                "tracks": ["return"],
            },
            "join": {
                "signature": "shlex.join(split_command: Iterable[str]) -> str",
                "description": "Joins tokens with proper shell quoting.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
        },
        "rules_using": [],
    },

    # =====================================================================
    # Stdlib — XML
    # =====================================================================

    "PyXmlEtree": {
        "description": (
            "xml.etree.ElementTree is the stdlib XML parser. The C-accelerated parser has some "
            "built-in protections but still processes external entities in some configurations — "
            "XXE sink. Prefer defusedxml for untrusted XML."
        ),
        "category": "deserialization",
        "fqns": ["xml.etree.ElementTree"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "parse": {
                "signature": "xml.etree.ElementTree.parse(source, parser=None) -> ElementTree",
                "description": "Parses an XML file. XXE sink under certain Python versions / custom parsers.",
                "role": "sink",
                "tracks": [0],
            },
            "fromstring": {
                "signature": "xml.etree.ElementTree.fromstring(text, parser=None) -> Element",
                "description": "Parses XML from a string. Same XXE considerations as parse().",
                "role": "sink",
                "tracks": [0],
            },
            "XMLParser": {
                "signature": "xml.etree.ElementTree.XMLParser(*, target=None, encoding=None) -> XMLParser",
                "description": "Custom parser. Pair with untrusted input to produce XXE.",
                "role": "sink",
                "tracks": [],
            },
        },
        "rules_using": [],
    },

    "PyXmlSax": {
        "description": (
            "xml.sax is the stdlib SAX parser. By default it resolves external entities — XXE "
            "sink on untrusted XML. Disable with parser.setFeature(feature_external_ges, False) "
            "or use defusedxml.sax."
        ),
        "category": "deserialization",
        "fqns": ["xml.sax"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "parse": {
                "signature": "xml.sax.parse(source, handler, error_handler=...) -> None",
                "description": "Parses XML with a SAX handler. XXE sink by default.",
                "role": "sink",
                "tracks": [0],
            },
            "parseString": {
                "signature": "xml.sax.parseString(string, handler, error_handler=...) -> None",
                "description": "Parses XML from a string. XXE sink.",
                "role": "sink",
                "tracks": [0],
            },
            "make_parser": {
                "signature": "xml.sax.make_parser(parser_list=()) -> XMLReader",
                "description": "Creates a SAX parser. XXE-prone unless external entities are disabled.",
                "role": "sink",
                "tracks": [],
            },
        },
        "rules_using": [],
    },

    "PyXmlDom": {
        "description": (
            "xml.dom.minidom for DOM-style XML parsing. Built on pyexpat which by default does "
            "not resolve external entities, but custom resolvers can reintroduce XXE. "
            "defusedxml.minidom is the hardened replacement."
        ),
        "category": "deserialization",
        "fqns": ["xml.dom", "xml.dom.minidom"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "parse": {
                "signature": "xml.dom.minidom.parse(file, parser=None, bufsize=None) -> Document",
                "description": "Parses XML file via minidom. XXE sink on custom parsers that resolve externals.",
                "role": "sink",
                "tracks": [0],
            },
            "parseString": {
                "signature": "xml.dom.minidom.parseString(string, parser=None) -> Document",
                "description": "Parses XML string. Same XXE considerations.",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": [],
    },

    "PyXmlrpc": {
        "description": (
            "xmlrpc.client and xmlrpc.server. ServerProxy RPCs execute arbitrary methods — "
            "dispatch on untrusted method names is a sink. ServerProxy + HTTP (not HTTPS) transmits "
            "credentials in plaintext."
        ),
        "category": "deserialization",
        "fqns": ["xmlrpc.client", "xmlrpc.server"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "ServerProxy": {
                "signature": "xmlrpc.client.ServerProxy(uri, transport=None, encoding=None, verbose=False, ...) -> ServerProxy",
                "description": "Opens an XML-RPC connection. Finding on http:// URIs (credentials in plaintext).",
                "role": "sink",
                "tracks": [0],
            },
            "loads": {
                "signature": "xmlrpc.client.loads(data, use_datetime=False, use_builtin_types=False) -> (params, methodname)",
                "description": "Parses an XML-RPC response. Inherits XXE surface from the XML parser.",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": [],
    },

    # =====================================================================
    # Stdlib — misc
    # =====================================================================

    "PyZlib": {
        "description": (
            "The zlib module for compression. decompress() on untrusted input can consume "
            "unbounded memory (zip bomb / decompression amplification). Set max_length to cap output."
        ),
        "category": "archives",
        "fqns": ["zlib"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "decompress": {
                "signature": "zlib.decompress(data: bytes, wbits=MAX_WBITS, bufsize=DEF_BUF_SIZE) -> bytes",
                "description": "Decompresses zlib / deflate data. Decompression-bomb sink on untrusted input without length cap.",
                "role": "sink",
                "tracks": [0],
            },
            "decompressobj": {
                "signature": "zlib.decompressobj(wbits=MAX_WBITS, zdict=b'') -> Decompress",
                "description": "Returns a streaming decompressor. Use with .decompress(data, max_length) to cap output.",
                "role": "neutral",
                "tracks": [],
            },
        },
        "rules_using": [],
    },

    "PyShelve": {
        "description": (
            "The shelve module persists arbitrary Python objects — backed by pickle under the "
            "hood. shelve.open() on untrusted files is a deserialization sink (RCE via pickle's "
            "__reduce__)."
        ),
        "category": "deserialization",
        "fqns": ["shelve"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "open": {
                "signature": "shelve.open(filename, flag='c', protocol=None, writeback=False) -> Shelf",
                "description": "Opens a shelf (pickle-backed dict). Deserialization sink on untrusted files.",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": [],
    },

    "PyLogging": {
        "description": (
            "The logging module. Most uses are neutral. Log-injection findings arise when "
            "user-controlled data is logged without sanitization — attackers can break log "
            "line boundaries with \\n or forge subsequent log entries."
        ),
        "category": "file-system",
        "fqns": ["logging"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "info": {
                "signature": "logging.info(msg, *args, **kwargs) -> None",
                "description": "Writes an info log. Log-injection sink when msg is tainted and contains newlines.",
                "role": "sink",
                "tracks": [0],
            },
            "error": {
                "signature": "logging.error(msg, *args, **kwargs) -> None",
                "description": "Writes an error log. Log-injection sink.",
                "role": "sink",
                "tracks": [0],
            },
            "warning": {
                "signature": "logging.warning(msg, *args, **kwargs) -> None",
                "description": "Writes a warning log. Log-injection sink.",
                "role": "sink",
                "tracks": [0],
            },
            "debug": {
                "signature": "logging.debug(msg, *args, **kwargs) -> None",
                "description": "Writes a debug log. Log-injection sink.",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": [],
    },

    # =====================================================================
    # Third-party — databases
    # =====================================================================

    "PySqlalchemy": {
        "description": (
            "SQLAlchemy is the most-used Python ORM. The text() wrapper and raw execute() are "
            "SQL injection sinks when the SQL is built from user input. Core and ORM query APIs "
            "with bound parameters are safe."
        ),
        "category": "databases",
        "fqns": ["sqlalchemy", "sqlalchemy.engine.Engine", "sqlalchemy.orm.Session"],
        "pip_snippet": "pip install sqlalchemy",
        "methods": {
            "text": {
                "signature": "sqlalchemy.text(text: str) -> TextClause",
                "description": "Wraps a raw SQL string. SQL injection sink when text is built from user input without :bindparams.",
                "role": "sink",
                "tracks": [0],
            },
            "Engine.execute": {
                "signature": "Engine.execute(statement, *multiparams, **params) -> CursorResult",
                "description": "Executes a statement. Injection sink when statement is a raw string.",
                "role": "sink",
                "tracks": [0],
            },
            "Connection.execute": {
                "signature": "Connection.execute(statement, parameters=None, ...) -> CursorResult",
                "description": "Executes a statement. Injection sink when statement is a raw string without text() + bindparams.",
                "role": "sink",
                "tracks": [0],
            },
            "Session.execute": {
                "signature": "Session.execute(statement, params=None, ...) -> Result",
                "description": "Executes a statement. Injection sink on raw strings.",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": [],
    },

    "PyPymysql": {
        "description": (
            "PyMySQL is a pure-Python MySQL driver. Cursor.execute() accepts a raw query and "
            "parameter tuple — injection sink when the query is built from user input without "
            "the %s placeholder."
        ),
        "category": "databases",
        "fqns": ["pymysql", "pymysql.cursors.Cursor"],
        "pip_snippet": "pip install pymysql",
        "methods": {
            "connect": {
                "signature": "pymysql.connect(host='localhost', user=None, password='', ...) -> Connection",
                "description": "Opens a MySQL connection.",
                "role": "neutral",
                "tracks": [],
            },
            "execute": {
                "signature": "Cursor.execute(query: str, args=None) -> int",
                "description": "Executes a query. SQL injection sink when query is built from user input without %s.",
                "role": "sink",
                "tracks": [0],
            },
            "executemany": {
                "signature": "Cursor.executemany(query: str, args: Sequence) -> int",
                "description": "Executes a query many times. Same injection risk.",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": [],
    },

    "PyMysqlDb": {
        "description": (
            "MySQLdb (mysqlclient) is a C-extension MySQL driver. Cursor.execute() is an SQL "
            "injection sink when the query is built without %s placeholders."
        ),
        "category": "databases",
        "fqns": ["MySQLdb", "MySQLdb.cursors.Cursor"],
        "pip_snippet": "pip install mysqlclient",
        "methods": {
            "connect": {
                "signature": "MySQLdb.connect(host='localhost', user=None, passwd='', ...) -> Connection",
                "description": "Opens a MySQL connection.",
                "role": "neutral",
                "tracks": [],
            },
            "execute": {
                "signature": "Cursor.execute(query: str, args=None) -> int",
                "description": "Executes a query. SQL injection sink.",
                "role": "sink",
                "tracks": [0],
            },
            "executemany": {
                "signature": "Cursor.executemany(query: str, args: Sequence) -> int",
                "description": "Batched query execution. Same injection risk.",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": [],
    },

    # =====================================================================
    # Third-party — HTTP
    # =====================================================================

    "PyHttpx": {
        "description": (
            "httpx is a modern async-capable HTTP client. Identical SSRF surface to requests — "
            "the URL argument on get/post/etc is a sink when user-controlled. verify=False "
            "disables TLS verification (separate rule)."
        ),
        "category": "http-clients",
        "fqns": ["httpx", "httpx.Client", "httpx.AsyncClient"],
        "pip_snippet": "pip install httpx",
        "methods": {
            "get": {
                "signature": "httpx.get(url, *, params=None, headers=None, ...) -> Response",
                "description": "Sends a GET request. SSRF sink on url.",
                "role": "sink",
                "tracks": [0],
            },
            "post": {
                "signature": "httpx.post(url, *, content=None, data=None, json=None, ...) -> Response",
                "description": "Sends a POST request. SSRF sink on url.",
                "role": "sink",
                "tracks": [0],
            },
            "put": {
                "signature": "httpx.put(url, *, content=None, data=None, ...) -> Response",
                "description": "Sends a PUT request. SSRF sink.",
                "role": "sink",
                "tracks": [0],
            },
            "delete": {
                "signature": "httpx.delete(url, ...) -> Response",
                "description": "Sends a DELETE request. SSRF sink.",
                "role": "sink",
                "tracks": [0],
            },
            "stream": {
                "signature": "httpx.stream(method, url, ...) -> ContextManager[Response]",
                "description": "Streams a response. SSRF sink on url.",
                "role": "sink",
                "tracks": [1],
            },
        },
        "rules_using": [],
    },

    "PyHttplib2": {
        "description": (
            "httplib2 is an HTTP client with advanced caching features. Http.request() is an "
            "SSRF sink when the URI is user-controlled."
        ),
        "category": "http-clients",
        "fqns": ["httplib2", "httplib2.Http"],
        "pip_snippet": "pip install httplib2",
        "methods": {
            "Http": {
                "signature": "httplib2.Http(cache=None, timeout=None, proxy_info=..., ca_certs=None, disable_ssl_certificate_validation=False) -> Http",
                "description": "Creates an HTTP client. Finding when disable_ssl_certificate_validation=True.",
                "role": "neutral",
                "tracks": [],
            },
            "request": {
                "signature": "Http.request(uri, method='GET', body=None, headers=None, ...) -> (Response, bytes)",
                "description": "Sends an HTTP request. SSRF sink on uri.",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": [],
    },

    # =====================================================================
    # Third-party — XML / parsing
    # =====================================================================

    "PyLxml": {
        "description": (
            "lxml is the C-backed XML / HTML parser. etree.parse / fromstring with a custom "
            "XMLParser(resolve_entities=True) is an XXE sink. Default behavior in recent lxml is "
            "safer but the API still allows unsafe configurations."
        ),
        "category": "deserialization",
        "fqns": ["lxml", "lxml.etree"],
        "pip_snippet": "pip install lxml",
        "methods": {
            "parse": {
                "signature": "lxml.etree.parse(source, parser=None, base_url=None) -> ElementTree",
                "description": "Parses XML. XXE sink when parser has resolve_entities=True.",
                "role": "sink",
                "tracks": [0],
            },
            "fromstring": {
                "signature": "lxml.etree.fromstring(text, parser=None, base_url=None) -> Element",
                "description": "Parses XML from string. XXE sink under unsafe parser config.",
                "role": "sink",
                "tracks": [0],
            },
            "XMLParser": {
                "signature": "lxml.etree.XMLParser(resolve_entities=False, no_network=True, ...) -> XMLParser",
                "description": "Creates an XML parser. Finding when resolve_entities=True or no_network=False.",
                "role": "neutral",
                "tracks": [],
            },
            "HTMLParser": {
                "signature": "lxml.etree.HTMLParser(recover=True, ...) -> HTMLParser",
                "description": "HTML parser variant. Less XXE risk than XML but still processes entities.",
                "role": "neutral",
                "tracks": [],
            },
        },
        "rules_using": [],
    },

    "PyDefusedXml": {
        "description": (
            "defusedxml is the hardened XML parser suite. It wraps xml.etree, xml.sax, xml.dom, "
            "lxml etc. with external-entity resolution disabled. Using defusedxml counterparts "
            "is the recommended sanitizer for XML sources."
        ),
        "category": "deserialization",
        "fqns": ["defusedxml"],
        "pip_snippet": "pip install defusedxml",
        "methods": {
            "parse": {
                "signature": "defusedxml.ElementTree.parse(source, parser=None) -> ElementTree",
                "description": "Safe XML parse. XXE-free. Sanitizer.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
            "fromstring": {
                "signature": "defusedxml.ElementTree.fromstring(text, parser=None) -> Element",
                "description": "Safe XML parse from string. Sanitizer.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
        },
        "rules_using": [],
    },

    # =====================================================================
    # Third-party — auth / crypto
    # =====================================================================

    "PyJose": {
        "description": (
            "python-jose implements JWT / JWS / JWE. jwt.decode() is the canonical validation "
            "entry point. Finding when algorithms=['none'] is passed (unsigned token acceptance) or "
            "verify_signature=False."
        ),
        "category": "crypto",
        "fqns": ["jose", "jose.jwt"],
        "pip_snippet": "pip install python-jose",
        "methods": {
            "encode": {
                "signature": "jose.jwt.encode(claims, key, algorithm='HS256', ...) -> str",
                "description": "Signs a JWT. Safe with a proper algorithm.",
                "role": "neutral",
                "tracks": [],
            },
            "decode": {
                "signature": "jose.jwt.decode(token, key, algorithms=None, options=None, ...) -> dict",
                "description": "Verifies and decodes a JWT. Finding when algorithms contains 'none' or options disable verification.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
            "get_unverified_header": {
                "signature": "jose.jwt.get_unverified_header(token) -> dict",
                "description": "Reads the JWT header without verifying the signature. Finding when return value drives auth decisions.",
                "role": "source",
                "tracks": ["return"],
            },
            "get_unverified_claims": {
                "signature": "jose.jwt.get_unverified_claims(token) -> dict",
                "description": "Reads claims without verifying. Finding for authz code.",
                "role": "source",
                "tracks": ["return"],
            },
        },
        "rules_using": [],
    },

    "PyAuthlib": {
        "description": (
            "Authlib is a comprehensive OAuth / OpenID / JWT library. JsonWebToken.decode() and "
            "the OAuth client Client.parse_request_body_response track access-token flows."
        ),
        "category": "crypto",
        "fqns": ["authlib", "authlib.jose"],
        "pip_snippet": "pip install authlib",
        "methods": {
            "jwt.encode": {
                "signature": "authlib.jose.jwt.encode(header, payload, key, check=True) -> bytes",
                "description": "Signs a JWT. Neutral with safe algorithm.",
                "role": "neutral",
                "tracks": [],
            },
            "jwt.decode": {
                "signature": "authlib.jose.jwt.decode(s, key, claims_cls=..., claims_options=..., ...) -> JWTClaims",
                "description": "Verifies and decodes a JWT. Finding under permissive claims_options.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
        },
        "rules_using": [],
    },

    "PyParamiko": {
        "description": (
            "paramiko is the SSH / SFTP client for Python. SSHClient.set_missing_host_key_policy "
            "with AutoAddPolicy() silently trusts unknown hosts — MITM risk. exec_command() is a "
            "command-execution sink when the command is user-controlled."
        ),
        "category": "crypto",
        "fqns": ["paramiko", "paramiko.SSHClient"],
        "pip_snippet": "pip install paramiko",
        "methods": {
            "SSHClient": {
                "signature": "paramiko.SSHClient() -> SSHClient",
                "description": "Creates an SSH client.",
                "role": "neutral",
                "tracks": [],
            },
            "set_missing_host_key_policy": {
                "signature": "SSHClient.set_missing_host_key_policy(policy: MissingHostKeyPolicy)",
                "description": "Sets host-key policy. Finding when policy is AutoAddPolicy() or WarningPolicy() (MITM).",
                "role": "sink",
                "tracks": [0],
            },
            "connect": {
                "signature": "SSHClient.connect(hostname, port=22, username=None, password=None, ...) -> None",
                "description": "Connects to an SSH server. SSRF-like sink when hostname is user-controlled.",
                "role": "sink",
                "tracks": [0],
            },
            "exec_command": {
                "signature": "SSHClient.exec_command(command: str, bufsize=-1, ...) -> (stdin, stdout, stderr)",
                "description": "Runs a command on the remote host. Command-injection sink on user-controlled command.",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": [],
    },

    "PyLdap3": {
        "description": (
            "ldap3 is a pure-Python LDAP client. Connection.search() accepts a search_filter — "
            "LDAP injection sink when the filter is built from user input without escaping. "
            "Use ldap3.utils.conv.escape_filter_chars() for safe construction."
        ),
        "category": "databases",
        "fqns": ["ldap3", "ldap3.Connection"],
        "pip_snippet": "pip install ldap3",
        "methods": {
            "Connection": {
                "signature": "ldap3.Connection(server, user=None, password=None, ...) -> Connection",
                "description": "Creates an LDAP connection.",
                "role": "neutral",
                "tracks": [],
            },
            "search": {
                "signature": "Connection.search(search_base, search_filter, search_scope=SUBTREE, ...) -> bool",
                "description": "Runs an LDAP search. Injection sink when search_filter is built from user input.",
                "role": "sink",
                "tracks": [1],
            },
            "bind": {
                "signature": "Connection.bind() -> bool",
                "description": "Binds / authenticates. Neutral.",
                "role": "neutral",
                "tracks": [],
            },
        },
        "rules_using": [],
    },

    # =====================================================================
    # Third-party — misc
    # =====================================================================

    "PyBleach": {
        "description": (
            "bleach is an HTML sanitizer library. bleach.clean() strips dangerous tags and "
            "attributes — sanitizer for XSS flows. bleach.linkify() is also safe."
        ),
        "category": "templating",
        "fqns": ["bleach"],
        "pip_snippet": "pip install bleach",
        "methods": {
            "clean": {
                "signature": "bleach.clean(text, tags=..., attributes=..., styles=..., ...) -> str",
                "description": "Strips dangerous HTML from text. XSS sanitizer.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
            "linkify": {
                "signature": "bleach.linkify(text, callbacks=..., skip_tags=None, parse_email=False) -> str",
                "description": "Converts URLs to safe <a> tags. Sanitizer.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
        },
        "rules_using": [],
    },

    "PyWerkzeug": {
        "description": (
            "Werkzeug is the WSGI toolkit Flask is built on. safe_join() is the canonical "
            "path-traversal sanitizer for serving files. utils.redirect is where Flask's open-redirect "
            "surface originates."
        ),
        "category": "web-frameworks",
        "fqns": ["werkzeug"],
        "pip_snippet": "pip install werkzeug",
        "methods": {
            "safe_join": {
                "signature": "werkzeug.utils.safe_join(directory: str, *pathnames) -> str | None",
                "description": "Safely joins a base directory with user-supplied components. Path-traversal sanitizer.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
            "redirect": {
                "signature": "werkzeug.utils.redirect(location: str, code: int = 302, Response=None) -> Response",
                "description": "Returns a redirect response. Open-redirect sink on user-controlled location.",
                "role": "sink",
                "tracks": [0],
            },
            "send_file": {
                "signature": "werkzeug.utils.send_file(path_or_file, environ, mimetype=None, ...) -> Response",
                "description": "Serves a file. Path-traversal sink on user-controlled path.",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": [],
    },

    "PyJsonschema": {
        "description": (
            "jsonschema validates JSON documents against a schema. validate() is a sanitizer for "
            "shape-checking untrusted JSON before passing fields to other sinks."
        ),
        "category": "web-frameworks",
        "fqns": ["jsonschema"],
        "pip_snippet": "pip install jsonschema",
        "methods": {
            "validate": {
                "signature": "jsonschema.validate(instance, schema, cls=None, ...) -> None",
                "description": "Raises ValidationError on mismatch. Sanitizer when it passes.",
                "role": "sanitizer",
                "tracks": [0],
            },
        },
        "rules_using": [],
    },

    "PyPydantic": {
        "description": (
            "Pydantic provides strict type-validated models. BaseModel parses / coerces input and "
            "raises on mismatch — the parsed model is a sanitizer for the raw input. Still, string "
            "fields on the model can remain tainted (not magically escaped)."
        ),
        "category": "web-frameworks",
        "fqns": ["pydantic", "pydantic.BaseModel"],
        "pip_snippet": "pip install pydantic",
        "methods": {
            "BaseModel": {
                "signature": "pydantic.BaseModel(**data: Any)",
                "description": "Constructs a validated model. Sanitizer for type / shape. String fields remain tainted.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
            "parse_obj": {
                "signature": "BaseModel.parse_obj(obj: Any) -> BaseModel",
                "description": "Parses a dict into a model. Sanitizer for shape.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
            "parse_raw": {
                "signature": "BaseModel.parse_raw(b: str | bytes, ...) -> BaseModel",
                "description": "Parses JSON / bytes into a model. Sanitizer.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
        },
        "rules_using": [],
    },

    "PyBoto3": {
        "description": (
            "boto3 is the AWS SDK for Python. client('s3').get_object(...) and similar operations "
            "commonly ingest user input into bucket / key names — SSRF-like vectors through S3 URLs "
            "and IAM misconfiguration. Covering for rule writers that check AWS-specific patterns."
        ),
        "category": "http-clients",
        "fqns": ["boto3"],
        "pip_snippet": "pip install boto3",
        "methods": {
            "client": {
                "signature": "boto3.client(service_name, region_name=None, ...) -> BaseClient",
                "description": "Creates a service client.",
                "role": "neutral",
                "tracks": [],
            },
            "resource": {
                "signature": "boto3.resource(service_name, region_name=None, ...) -> ServiceResource",
                "description": "Creates a higher-level resource client.",
                "role": "neutral",
                "tracks": [],
            },
        },
        "rules_using": [],
    },

    "PyAiofiles": {
        "description": (
            "aiofiles provides async file I/O. aiofiles.open() is a path-traversal sink when "
            "the path is user-controlled (same as built-in open)."
        ),
        "category": "file-system",
        "fqns": ["aiofiles"],
        "pip_snippet": "pip install aiofiles",
        "methods": {
            "open": {
                "signature": "aiofiles.open(file, mode='r', buffering=-1, encoding=None, ...) -> AsyncTextIOWrapper",
                "description": "Async file open. Path-traversal sink on user-controlled file path.",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": [],
    },

    # =====================================================================
    # Stdlib — HTML / escaping / templating
    # =====================================================================

    "PyHtml": {
        "description": (
            "The html module. html.escape() is the canonical XSS sanitizer for writing user "
            "input into HTML text content. html.unescape() does the inverse and should NOT be "
            "used on output paths."
        ),
        "category": "templating",
        "fqns": ["html", "html.parser"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "escape": {
                "signature": "html.escape(s: str, quote: bool = True) -> str",
                "description": "Escapes &, <, > and optionally \" and ' for HTML text. XSS sanitizer.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
            "unescape": {
                "signature": "html.unescape(s: str) -> str",
                "description": "Converts HTML entities back to chars. Inverse of escape(). Not a sanitizer.",
                "role": "neutral",
                "tracks": [],
            },
        },
        "rules_using": [],
    },

    "PyStringTemplate": {
        "description": (
            "string.Template and string.Formatter. Template($var) substitution is safe when "
            "placeholders are explicit. Formatter.format() with user-controlled format_spec is "
            "a format-string injection vector."
        ),
        "category": "templating",
        "fqns": ["string"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "Template": {
                "signature": "string.Template(template: str) -> Template",
                "description": "Creates a $-substitution template. Neutral; substitute() is the rendering step.",
                "role": "neutral",
                "tracks": [],
            },
            "Formatter": {
                "signature": "string.Formatter() -> Formatter",
                "description": "Advanced str.format interface. Format-string injection sink when format string is user-controlled.",
                "role": "sink",
                "tracks": [],
            },
        },
        "rules_using": [],
    },

    # =====================================================================
    # Stdlib — regex / validation
    # =====================================================================

    "PyRe": {
        "description": (
            "The re module. Catastrophic backtracking in regex patterns (ReDoS) — finding when a "
            "user-controlled pattern flows into re.compile / re.search / re.match. Also, re.findall "
            "on untrusted HTML is a common anti-pattern that misses cases."
        ),
        "category": "file-system",
        "fqns": ["re"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "compile": {
                "signature": "re.compile(pattern, flags=0) -> Pattern",
                "description": "Compiles a regex. ReDoS sink when pattern is user-controlled.",
                "role": "sink",
                "tracks": [0],
            },
            "match": {
                "signature": "re.match(pattern, string, flags=0) -> Match | None",
                "description": "Matches at start of string. ReDoS sink on user-controlled pattern.",
                "role": "sink",
                "tracks": [0],
            },
            "search": {
                "signature": "re.search(pattern, string, flags=0) -> Match | None",
                "description": "Searches for pattern. ReDoS sink on user-controlled pattern.",
                "role": "sink",
                "tracks": [0],
            },
            "sub": {
                "signature": "re.sub(pattern, repl, string, count=0, flags=0) -> str",
                "description": "Regex-based substitution. ReDoS sink on user-controlled pattern.",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": [],
    },

    "PyIpaddress": {
        "description": (
            "The ipaddress module for IP address parsing and classification. IPv4Address / "
            "IPv6Address constructors raise on invalid input — sanitizer for IP flows. "
            "is_private / is_loopback / is_reserved are building blocks for SSRF defense."
        ),
        "category": "http-clients",
        "fqns": ["ipaddress"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "ip_address": {
                "signature": "ipaddress.ip_address(address) -> IPv4Address | IPv6Address",
                "description": "Parses an IP address. Sanitizer (raises on invalid input).",
                "role": "sanitizer",
                "tracks": ["return"],
            },
            "ip_network": {
                "signature": "ipaddress.ip_network(address, strict=True) -> IPv4Network | IPv6Network",
                "description": "Parses an IP network. Sanitizer.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
        },
        "rules_using": [],
    },

    # =====================================================================
    # Stdlib — CSV / email / config
    # =====================================================================

    "PyCsv": {
        "description": (
            "The csv module. csv.writer + writerow on user-controlled cells produces CSV-formula "
            "injection when the receiver opens the CSV in Excel (cells starting with =, +, -, @ are "
            "interpreted as formulas). No stdlib sanitizer — prefix with a tab or apostrophe."
        ),
        "category": "deserialization",
        "fqns": ["csv"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "writer": {
                "signature": "csv.writer(csvfile, dialect='excel', **fmtparams) -> _writer",
                "description": "Creates a CSV writer. writerow() with user-controlled cells is a formula-injection sink.",
                "role": "neutral",
                "tracks": [],
            },
            "reader": {
                "signature": "csv.reader(csvfile, dialect='excel', **fmtparams) -> _reader",
                "description": "Creates a CSV reader. Rows are sources when the file is user-supplied.",
                "role": "source",
                "tracks": ["return"],
            },
            "DictReader": {
                "signature": "csv.DictReader(f, fieldnames=None, ...) -> DictReader",
                "description": "CSV reader that maps rows to dicts. Source on untrusted CSV files.",
                "role": "source",
                "tracks": ["return"],
            },
            "DictWriter": {
                "signature": "csv.DictWriter(f, fieldnames, ...) -> DictWriter",
                "description": "Dict-based CSV writer. Formula-injection sink on user cells.",
                "role": "neutral",
                "tracks": [],
            },
        },
        "rules_using": [],
    },

    "PyEmail": {
        "description": (
            "The email package. email.message.EmailMessage assembly with user-controlled Subject, "
            "To, From, or body is an email-header-injection sink (CRLF in header values can inject "
            "extra headers). email.parser handles incoming messages — sources of user content."
        ),
        "category": "http-clients",
        "fqns": ["email", "email.message", "email.parser", "email.mime"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "EmailMessage": {
                "signature": "email.message.EmailMessage(policy=default) -> EmailMessage",
                "description": "Creates a message. Setting headers from user input is a CRLF-injection sink.",
                "role": "neutral",
                "tracks": [],
            },
            "message_from_string": {
                "signature": "email.message_from_string(s, _class=EmailMessage, *, policy=compat32) -> EmailMessage",
                "description": "Parses a message from a string. Source for incoming email content.",
                "role": "source",
                "tracks": ["return"],
            },
            "message_from_bytes": {
                "signature": "email.message_from_bytes(s, _class=EmailMessage, *, policy=compat32) -> EmailMessage",
                "description": "Parses a message from bytes. Source.",
                "role": "source",
                "tracks": ["return"],
            },
        },
        "rules_using": [],
    },

    "PyConfigparser": {
        "description": (
            "The configparser module reads INI-style config files. Values read via get() are "
            "sources when the config file is user-supplied. The module itself has no injection "
            "sinks of its own."
        ),
        "category": "file-system",
        "fqns": ["configparser"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "ConfigParser": {
                "signature": "configparser.ConfigParser(defaults=None, ...) -> ConfigParser",
                "description": "Creates a parser.",
                "role": "neutral",
                "tracks": [],
            },
            "read": {
                "signature": "ConfigParser.read(filenames, encoding=None) -> list[str]",
                "description": "Reads config files. Subsequent get() values become sources.",
                "role": "neutral",
                "tracks": [],
            },
            "get": {
                "signature": "ConfigParser.get(section: str, option: str, *, raw=False, vars=None, fallback=...) -> str",
                "description": "Returns a config value. Source when the config file is user-supplied.",
                "role": "source",
                "tracks": ["return"],
            },
        },
        "rules_using": [],
    },

    # =====================================================================
    # Stdlib — HTTP server / cookies / IMAP / POP3 / CGI / WSGI
    # =====================================================================

    "PyHttpServer": {
        "description": (
            "The http.server module. SimpleHTTPRequestHandler serves files from the current "
            "working directory — path-traversal sink on directory containing secrets. Intended "
            "for development only, finding on any production use."
        ),
        "category": "http-clients",
        "fqns": ["http.server"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "HTTPServer": {
                "signature": "http.server.HTTPServer(server_address, RequestHandlerClass, bind_and_activate=True) -> HTTPServer",
                "description": "HTTP server. Finding when bound to 0.0.0.0 without access control.",
                "role": "sink",
                "tracks": [],
            },
            "SimpleHTTPRequestHandler": {
                "signature": "http.server.SimpleHTTPRequestHandler(*args, **kwargs) -> SimpleHTTPRequestHandler",
                "description": "Serves files from CWD. Path-traversal sink for sensitive directories.",
                "role": "sink",
                "tracks": [],
            },
        },
        "rules_using": [],
    },

    "PyHttpCookies": {
        "description": (
            "The http.cookies module for cookie parsing. SimpleCookie accepts raw Cookie headers "
            "— the parsed morsels carry user input. Setting a cookie without Secure / HttpOnly / "
            "SameSite is a common hardening finding."
        ),
        "category": "http-clients",
        "fqns": ["http.cookies"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "SimpleCookie": {
                "signature": "http.cookies.SimpleCookie(input=None) -> SimpleCookie",
                "description": "Parses a Cookie header. Parsed morsels are sources.",
                "role": "source",
                "tracks": ["return"],
            },
            "Morsel": {
                "signature": "http.cookies.Morsel() -> Morsel",
                "description": "Represents one cookie. Finding when secure/httponly/samesite flags are not set.",
                "role": "neutral",
                "tracks": [],
            },
        },
        "rules_using": [],
    },

    "PyImaplib": {
        "description": (
            "The imaplib module. IMAP4() uses plaintext; IMAP4_SSL is the encrypted variant. "
            "Any use of plain IMAP is a credential-over-plaintext finding."
        ),
        "category": "http-clients",
        "fqns": ["imaplib"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "IMAP4": {
                "signature": "imaplib.IMAP4(host='', port=143, timeout=None) -> IMAP4",
                "description": "Plaintext IMAP. Credentials transmitted unencrypted. Finding.",
                "role": "sink",
                "tracks": [],
            },
            "IMAP4_SSL": {
                "signature": "imaplib.IMAP4_SSL(host='', port=993, *, ssl_context=None, timeout=None) -> IMAP4_SSL",
                "description": "IMAP over TLS. Safe.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
        },
        "rules_using": [],
    },

    "PyPoplib": {
        "description": (
            "The poplib module. POP3() is plaintext; POP3_SSL encrypts. Plaintext POP3 is a "
            "credential-over-plaintext finding."
        ),
        "category": "http-clients",
        "fqns": ["poplib"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "POP3": {
                "signature": "poplib.POP3(host, port=110, timeout=...) -> POP3",
                "description": "Plaintext POP3. Finding.",
                "role": "sink",
                "tracks": [],
            },
            "POP3_SSL": {
                "signature": "poplib.POP3_SSL(host, port=995, keyfile=None, certfile=None, timeout=..., context=None) -> POP3_SSL",
                "description": "POP3 over TLS. Safe.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
        },
        "rules_using": [],
    },

    "PyCgi": {
        "description": (
            "The cgi module (deprecated in 3.11, removed in 3.13). cgi.FieldStorage collects form "
            "data for CGI scripts — each field value is a source. Any new code should not use cgi."
        ),
        "category": "web-frameworks",
        "fqns": ["cgi"],
        "pip_snippet": "# stdlib — deprecated in Python 3.11, removed in 3.13",
        "methods": {
            "FieldStorage": {
                "signature": "cgi.FieldStorage(fp=None, headers=None, outerboundary=b'', environ=os.environ, ...) -> FieldStorage",
                "description": "Parses form data. Each field is user-controlled.",
                "role": "source",
                "tracks": ["return"],
            },
            "parse": {
                "signature": "cgi.parse(fp=None, environ=os.environ, keep_blank_values=False, strict_parsing=False, separator='&') -> dict",
                "description": "Parses form data into a dict. Source.",
                "role": "source",
                "tracks": ["return"],
            },
        },
        "rules_using": [],
    },

    "PyWsgiref": {
        "description": (
            "The wsgiref module for WSGI utilities. simple_server.make_server is dev-only — "
            "production should use gunicorn or waitress. util.request_uri reconstructs the URL "
            "from environ and is a source."
        ),
        "category": "web-frameworks",
        "fqns": ["wsgiref"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "make_server": {
                "signature": "wsgiref.simple_server.make_server(host, port, app, ...) -> WSGIServer",
                "description": "Creates a development WSGI server. Finding on production use.",
                "role": "sink",
                "tracks": [],
            },
            "request_uri": {
                "signature": "wsgiref.util.request_uri(environ, include_query=True) -> str",
                "description": "Reconstructs the request URL. Source when environ reflects real traffic.",
                "role": "source",
                "tracks": ["return"],
            },
        },
        "rules_using": [],
    },

    "PyGetpass": {
        "description": (
            "The getpass module. getpass.getpass() prompts for a password without echoing. "
            "getpass.getuser() returns the current user — source when used for authorization decisions."
        ),
        "category": "crypto",
        "fqns": ["getpass"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "getpass": {
                "signature": "getpass.getpass(prompt='Password: ', stream=None) -> str",
                "description": "Prompts for password. Source (user-controlled).",
                "role": "source",
                "tracks": ["return"],
            },
            "getuser": {
                "signature": "getpass.getuser() -> str",
                "description": "Returns the current login name. Source when used for access checks (env variables can spoof).",
                "role": "source",
                "tracks": ["return"],
            },
        },
        "rules_using": [],
    },

    "PyGlob": {
        "description": (
            "The glob module. glob.glob() resolves shell-style patterns against the filesystem — "
            "finding when the pattern is user-controlled (can enumerate directories outside intended scope)."
        ),
        "category": "file-system",
        "fqns": ["glob"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "glob": {
                "signature": "glob.glob(pathname, *, root_dir=None, dir_fd=None, recursive=False, ...) -> list[str]",
                "description": "Returns matching paths. Finding when pathname is user-controlled.",
                "role": "sink",
                "tracks": [0],
            },
            "iglob": {
                "signature": "glob.iglob(pathname, *, root_dir=None, ...) -> Iterator[str]",
                "description": "Like glob() but returns an iterator. Same risk.",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": [],
    },

    # =====================================================================
    # Stdlib — FFI / crypto legacy
    # =====================================================================

    "PyCtypes": {
        "description": (
            "The ctypes module for calling C libraries. LoadLibrary / CDLL on user-controlled "
            "paths loads arbitrary code — code-execution sink. String pointer operations can "
            "also be memory-safety findings."
        ),
        "category": "command-execution",
        "fqns": ["ctypes"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "CDLL": {
                "signature": "ctypes.CDLL(name, mode=DEFAULT_MODE, handle=None, use_errno=False, use_last_error=False, winmode=None) -> CDLL",
                "description": "Loads a shared library. Code-execution sink on user-controlled name.",
                "role": "sink",
                "tracks": [0],
            },
            "WinDLL": {
                "signature": "ctypes.WinDLL(name, ...) -> WinDLL",
                "description": "Windows shared library loader. Code-execution sink.",
                "role": "sink",
                "tracks": [0],
            },
            "LoadLibrary": {
                "signature": "ctypes.cdll.LoadLibrary(name) -> CDLL",
                "description": "Loads a shared library. Code-execution sink on user-controlled name.",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": [],
    },

    "PyCrypt": {
        "description": (
            "The crypt module (deprecated in 3.11, removed in 3.13). crypt.crypt() wraps the "
            "Unix crypt(3) call. Most default methods are weak (DES, MD5). Use passlib or "
            "hashlib.scrypt / pbkdf2_hmac instead."
        ),
        "category": "crypto",
        "fqns": ["crypt"],
        "pip_snippet": "# stdlib — deprecated in 3.11, removed in 3.13. Use passlib.",
        "methods": {
            "crypt": {
                "signature": "crypt.crypt(word, salt=None) -> str",
                "description": "Unix crypt password hashing. Finding — algorithm selection is platform-dependent and often weak.",
                "role": "sink",
                "tracks": [],
            },
        },
        "rules_using": [],
    },

    "PyDbm": {
        "description": (
            "The dbm family (dbm.gnu, dbm.ndbm, dbm.dumb). dbm.open() on untrusted files reads "
            "a DBM-format database. dbm.dumb is pickle-like and unsafe on untrusted input."
        ),
        "category": "deserialization",
        "fqns": ["dbm"],
        "pip_snippet": "# stdlib — no install required",
        "methods": {
            "open": {
                "signature": "dbm.open(file, flag='r', mode=0o666) -> dbm",
                "description": "Opens a DBM database. Finding on untrusted files (dbm.dumb is especially unsafe).",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": [],
    },

    # =====================================================================
    # Third-party — web frameworks / servers
    # =====================================================================

    "PyAiohttp": {
        "description": (
            "aiohttp provides async HTTP client and server. ClientSession.get / post and the "
            "top-level request() are SSRF sinks on user-controlled URLs. aiohttp.web request "
            "handlers expose sources via request.query, request.post, request.json."
        ),
        "category": "http-clients",
        "fqns": ["aiohttp", "aiohttp.ClientSession", "aiohttp.web"],
        "pip_snippet": "pip install aiohttp",
        "methods": {
            "ClientSession.get": {
                "signature": "async ClientSession.get(url, *, allow_redirects=True, ...) -> ClientResponse",
                "description": "Async GET. SSRF sink on url.",
                "role": "sink",
                "tracks": [0],
            },
            "ClientSession.post": {
                "signature": "async ClientSession.post(url, *, data=None, json=None, ...) -> ClientResponse",
                "description": "Async POST. SSRF sink.",
                "role": "sink",
                "tracks": [0],
            },
            "request": {
                "signature": "async aiohttp.request(method, url, **kwargs) -> ClientResponse",
                "description": "Top-level async request helper. SSRF sink on url.",
                "role": "sink",
                "tracks": [1],
            },
            "Request.query": {
                "signature": "request.query: MultiDict[str, str]",
                "description": "URL query parameters on aiohttp.web handlers. User-controlled.",
                "role": "source",
                "tracks": ["return"],
            },
            "Request.post": {
                "signature": "async request.post() -> MultiDict[str, str]",
                "description": "Form body. User-controlled.",
                "role": "source",
                "tracks": ["return"],
            },
            "Request.json": {
                "signature": "async request.json(*, loads=json.loads) -> Any",
                "description": "Parsed JSON body. User-controlled.",
                "role": "source",
                "tracks": ["return"],
            },
        },
        "rules_using": [],
    },

    "PyStarlette": {
        "description": (
            "Starlette is the ASGI toolkit behind FastAPI. Request exposes HTTP input; the "
            "responses module provides HTMLResponse / RedirectResponse / FileResponse (sinks for "
            "XSS, open-redirect, path-traversal respectively)."
        ),
        "category": "web-frameworks",
        "fqns": ["starlette", "starlette.requests", "starlette.responses"],
        "pip_snippet": "pip install starlette",
        "methods": {
            "Request.query_params": {
                "signature": "request.query_params: QueryParams",
                "description": "URL query parameters.",
                "role": "source",
                "tracks": ["return"],
            },
            "Request.path_params": {
                "signature": "request.path_params: dict",
                "description": "Path parameters.",
                "role": "source",
                "tracks": ["return"],
            },
            "Request.form": {
                "signature": "async request.form() -> FormData",
                "description": "Form body.",
                "role": "source",
                "tracks": ["return"],
            },
            "HTMLResponse": {
                "signature": "HTMLResponse(content, status_code=200, headers=None, media_type=None, ...) -> Response",
                "description": "Raw HTML response. XSS sink on tainted content.",
                "role": "sink",
                "tracks": [0],
            },
            "RedirectResponse": {
                "signature": "RedirectResponse(url, status_code=307, ...) -> Response",
                "description": "Redirect response. Open-redirect sink.",
                "role": "sink",
                "tracks": [0],
            },
            "FileResponse": {
                "signature": "FileResponse(path, status_code=200, headers=None, media_type=None, filename=None, ...) -> Response",
                "description": "Serves a file. Path-traversal sink on user-controlled path.",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": [],
    },

    "PyRestFramework": {
        "description": (
            "Django REST Framework (DRF). request.data is the primary source for JSON / form "
            "payloads; serializers validate input (sanitizer when is_valid is called with "
            "raise_exception=True). Response() with tainted data is generally safe due to DRF's "
            "renderers but render_template is still worth watching."
        ),
        "category": "web-frameworks",
        "fqns": ["rest_framework", "rest_framework.request", "rest_framework.response"],
        "pip_snippet": "pip install djangorestframework",
        "methods": {
            "Request.data": {
                "signature": "request.data: dict | list",
                "description": "Parsed body (JSON, form, multipart). User-controlled.",
                "role": "source",
                "tracks": ["return"],
            },
            "Request.query_params": {
                "signature": "request.query_params: QueryDict",
                "description": "URL query parameters.",
                "role": "source",
                "tracks": ["return"],
            },
            "Serializer.is_valid": {
                "signature": "Serializer.is_valid(raise_exception=False) -> bool",
                "description": "Validates input. Sanitizer when raise_exception=True.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
            "Response": {
                "signature": "Response(data=None, status=None, template_name=None, headers=None, ...) -> Response",
                "description": "DRF response. Data is rendered safely; template_name can be an SSTI sink.",
                "role": "neutral",
                "tracks": [],
            },
        },
        "rules_using": [],
    },

    "PyFlaskCors": {
        "description": (
            "flask-cors configures CORS headers on Flask apps. CORS(app, origins='*') with "
            "supports_credentials=True is a major finding (wildcard origin with credentials is "
            "explicitly forbidden by browsers but some configurations still emit it)."
        ),
        "category": "web-frameworks",
        "fqns": ["flask_cors", "flask_cors.CORS"],
        "pip_snippet": "pip install flask-cors",
        "methods": {
            "CORS": {
                "signature": "CORS(app=None, *, resources=..., origins=None, supports_credentials=False, ...) -> CORS",
                "description": "Installs CORS headers. Finding when origins='*' and supports_credentials=True.",
                "role": "sink",
                "tracks": [],
            },
            "cross_origin": {
                "signature": "cross_origin(origins=None, methods=None, supports_credentials=False, ...) -> Callable",
                "description": "Per-view CORS decorator. Same credential wildcard finding applies.",
                "role": "sink",
                "tracks": [],
            },
        },
        "rules_using": [],
    },

    "PyWerkzeugSecurity": {
        "description": (
            "werkzeug.security provides generate_password_hash and check_password_hash. The "
            "default method is pbkdf2:sha256 with 600_000 iterations. Findings arise when "
            "method='plain' or a weak hasher is passed explicitly."
        ),
        "category": "crypto",
        "fqns": ["werkzeug.security"],
        "pip_snippet": "pip install werkzeug",
        "methods": {
            "generate_password_hash": {
                "signature": "werkzeug.security.generate_password_hash(password, method='scrypt', salt_length=16) -> str",
                "description": "Hashes a password. Safe with default method. Finding when method='plain'.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
            "check_password_hash": {
                "signature": "werkzeug.security.check_password_hash(pwhash: str, password: str) -> bool",
                "description": "Constant-time password verification. Safe.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
        },
        "rules_using": [],
    },

    # =====================================================================
    # Third-party — additional crypto / auth
    # =====================================================================

    "PyPyjwt": {
        "description": (
            "PyJWT decodes and validates JWTs. jwt.decode() with algorithms=['none'] or "
            "options={'verify_signature': False} accepts unsigned tokens — major finding. "
            "Always pass algorithms explicitly."
        ),
        "category": "crypto",
        "fqns": ["jwt"],
        "pip_snippet": "pip install pyjwt",
        "methods": {
            "encode": {
                "signature": "jwt.encode(payload: dict, key: str | bytes, algorithm='HS256', headers=None, json_encoder=None) -> str",
                "description": "Signs a JWT. Safe with a proper algorithm.",
                "role": "neutral",
                "tracks": [],
            },
            "decode": {
                "signature": "jwt.decode(jwt: str, key=None, algorithms=None, options=None, audience=None, issuer=None, leeway=0) -> dict",
                "description": "Verifies and decodes. Finding on algorithms=['none'] or verify_signature=False.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
        },
        "rules_using": [],
    },

    "PyOauthlib": {
        "description": (
            "oauthlib implements the OAuth 1 / OAuth 2 protocols. WebApplicationClient.parse_request_uri_response "
            "extracts the authorization code from the callback URL — source for subsequent token "
            "exchange."
        ),
        "category": "crypto",
        "fqns": ["oauthlib"],
        "pip_snippet": "pip install oauthlib",
        "methods": {
            "WebApplicationClient": {
                "signature": "oauthlib.oauth2.WebApplicationClient(client_id, ...) -> WebApplicationClient",
                "description": "OAuth 2 client.",
                "role": "neutral",
                "tracks": [],
            },
            "parse_request_uri_response": {
                "signature": "WebApplicationClient.parse_request_uri_response(uri, state=None) -> dict",
                "description": "Extracts code / tokens from callback URI. Source for tokens.",
                "role": "source",
                "tracks": ["return"],
            },
        },
        "rules_using": [],
    },

    "PyCryptography": {
        "description": (
            "The cryptography package provides recipes (Fernet) and primitives (hazmat). Fernet "
            "is the recommended symmetric encryption helper. Findings arise when hazmat primitives "
            "are used with obsolete algorithms (MD5, DES, RC4) or ECB mode."
        ),
        "category": "crypto",
        "fqns": ["cryptography", "cryptography.fernet", "cryptography.hazmat"],
        "pip_snippet": "pip install cryptography",
        "methods": {
            "Fernet": {
                "signature": "cryptography.fernet.Fernet(key: bytes) -> Fernet",
                "description": "Authenticated symmetric encryption. Safe.",
                "role": "sanitizer",
                "tracks": [],
            },
            "Fernet.encrypt": {
                "signature": "Fernet.encrypt(data: bytes) -> bytes",
                "description": "Encrypts a message. Safe.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
            "Fernet.decrypt": {
                "signature": "Fernet.decrypt(token: bytes, ttl: int = None) -> bytes",
                "description": "Decrypts and authenticates. Raises on tampering. Safe.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
            "Cipher": {
                "signature": "cryptography.hazmat.primitives.ciphers.Cipher(algorithm, mode, backend=None) -> Cipher",
                "description": "Low-level cipher. Finding when algorithm is DES/3DES/RC4 or mode is ECB.",
                "role": "neutral",
                "tracks": [],
            },
        },
        "rules_using": [],
    },

    "PyHvac": {
        "description": (
            "hvac is the Python client for HashiCorp Vault. Client.secrets.kv.v2.read_secret_version "
            "reads a secret — the returned payload is a source. Client() with verify=False disables "
            "TLS verification (major finding)."
        ),
        "category": "crypto",
        "fqns": ["hvac", "hvac.Client"],
        "pip_snippet": "pip install hvac",
        "methods": {
            "Client": {
                "signature": "hvac.Client(url='http://localhost:8200', token=None, verify=True, ...) -> Client",
                "description": "Vault client. Finding when verify=False.",
                "role": "neutral",
                "tracks": [],
            },
            "secrets.kv.v2.read_secret_version": {
                "signature": "Client.secrets.kv.v2.read_secret_version(path, mount_point='secret', version=None, ...) -> dict",
                "description": "Reads a KV secret. Return value carries secret data.",
                "role": "source",
                "tracks": ["return"],
            },
        },
        "rules_using": [],
    },

    # =====================================================================
    # Third-party — additional HTTP / sockets
    # =====================================================================

    "PyPycurl": {
        "description": (
            "pycurl wraps libcurl. curl.setopt(pycurl.URL, ...) is an SSRF sink on user-controlled "
            "URLs. setopt(pycurl.SSL_VERIFYPEER, 0) disables TLS verification."
        ),
        "category": "http-clients",
        "fqns": ["pycurl"],
        "pip_snippet": "pip install pycurl",
        "methods": {
            "Curl": {
                "signature": "pycurl.Curl() -> Curl",
                "description": "Creates a cURL handle.",
                "role": "neutral",
                "tracks": [],
            },
            "setopt": {
                "signature": "Curl.setopt(option, value) -> None",
                "description": "Sets a cURL option. SSRF sink when option=pycurl.URL and value is user-controlled.",
                "role": "sink",
                "tracks": [1],
            },
            "perform": {
                "signature": "Curl.perform() -> None",
                "description": "Sends the request. Sink in combination with setopt.",
                "role": "sink",
                "tracks": [],
            },
        },
        "rules_using": [],
    },

    "PyPysftp": {
        "description": (
            "pysftp wraps paramiko with a simpler SFTP interface. Connection(host, cnopts=...) "
            "with CnOpts.hostkeys=None disables host-key checking — MITM finding."
        ),
        "category": "crypto",
        "fqns": ["pysftp"],
        "pip_snippet": "pip install pysftp",
        "methods": {
            "Connection": {
                "signature": "pysftp.Connection(host, username=None, private_key=None, password=None, port=22, cnopts=None, ...) -> Connection",
                "description": "Opens an SFTP connection. Finding when cnopts.hostkeys is None.",
                "role": "sink",
                "tracks": [],
            },
            "put": {
                "signature": "Connection.put(localpath, remotepath=None, callback=None, confirm=True, preserve_mtime=False) -> SFTPAttributes",
                "description": "Uploads a file. Path-traversal risk on remotepath.",
                "role": "sink",
                "tracks": [0, 1],
            },
            "get": {
                "signature": "Connection.get(remotepath, localpath=None, callback=None, preserve_mtime=False) -> None",
                "description": "Downloads a file. Path-traversal risk on localpath.",
                "role": "sink",
                "tracks": [0, 1],
            },
        },
        "rules_using": [],
    },

    # =====================================================================
    # Third-party — infra / misc
    # =====================================================================

    "PyDocker": {
        "description": (
            "The docker SDK. DockerClient.containers.run with privileged=True is a container-"
            "escape finding. volumes mounting /var/run/docker.sock into the container grants full "
            "Docker daemon access."
        ),
        "category": "command-execution",
        "fqns": ["docker", "docker.DockerClient"],
        "pip_snippet": "pip install docker",
        "methods": {
            "from_env": {
                "signature": "docker.from_env(version=None, timeout=60, ...) -> DockerClient",
                "description": "Creates a client from DOCKER_HOST env vars.",
                "role": "neutral",
                "tracks": [],
            },
            "containers.run": {
                "signature": "Container.run(image, command=None, *, privileged=False, volumes=None, ...) -> Container",
                "description": "Runs a container. Finding when privileged=True or docker.sock is mounted.",
                "role": "sink",
                "tracks": [0, 1],
            },
        },
        "rules_using": [],
    },

    "PyCelery": {
        "description": (
            "Celery is a distributed task queue. Celery(broker=..., backend=...) configures brokers — "
            "findings when broker URL has insecure defaults (redis:// without TLS, amqp:// without TLS). "
            "@task decorators accept arbitrary user-controlled args via the queue."
        ),
        "category": "web-frameworks",
        "fqns": ["celery", "celery.Celery"],
        "pip_snippet": "pip install celery",
        "methods": {
            "Celery": {
                "signature": "celery.Celery(main=None, broker=None, backend=None, ...) -> Celery",
                "description": "Celery app. Finding when broker scheme is redis:// or amqp:// without TLS.",
                "role": "neutral",
                "tracks": [],
            },
            "task": {
                "signature": "@celery.task(bind=False, ...) -> Callable",
                "description": "Registers a task. Arguments are user-controlled sources.",
                "role": "source",
                "tracks": ["return"],
            },
        },
        "rules_using": [],
    },

    "PyWaitress": {
        "description": (
            "waitress is a production WSGI server. serve() with host='0.0.0.0' exposes the app "
            "to all interfaces — finding for internal-only services."
        ),
        "category": "web-frameworks",
        "fqns": ["waitress"],
        "pip_snippet": "pip install waitress",
        "methods": {
            "serve": {
                "signature": "waitress.serve(app, host='0.0.0.0', port=8080, ...) -> None",
                "description": "Serves a WSGI app. Finding when bound to 0.0.0.0 for internal apps.",
                "role": "sink",
                "tracks": [],
            },
        },
        "rules_using": [],
    },

    "PyGunicorn": {
        "description": (
            "gunicorn is a production WSGI server. Commonly run via CLI but programmatic use via "
            "Application() is possible. bind '0.0.0.0:*' on internal apps is a finding."
        ),
        "category": "web-frameworks",
        "fqns": ["gunicorn"],
        "pip_snippet": "pip install gunicorn",
        "methods": {
            "Application": {
                "signature": "gunicorn.app.base.BaseApplication() -> BaseApplication",
                "description": "Gunicorn application base class.",
                "role": "neutral",
                "tracks": [],
            },
        },
        "rules_using": [],
    },

    "PyPyasn1": {
        "description": (
            "pyasn1 decodes ASN.1 structures. der_decoder.decode() on untrusted DER bytes can "
            "trigger denial-of-service via deep nesting. Typically used in certificate / LDAP "
            "contexts."
        ),
        "category": "deserialization",
        "fqns": ["pyasn1"],
        "pip_snippet": "pip install pyasn1",
        "methods": {
            "der_decoder.decode": {
                "signature": "pyasn1.codec.der.decoder.decode(substrate, asn1Spec=None, ...) -> (value, rest)",
                "description": "Decodes DER bytes. Sink for malformed / nested input (DoS).",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": [],
    },

    "PyDockerfileParse": {
        "description": (
            "dockerfile_parse parses Dockerfiles. Returned structures reflect user-controlled "
            "file content. Usually a source for linting rules, not a sink."
        ),
        "category": "file-system",
        "fqns": ["dockerfile_parse"],
        "pip_snippet": "pip install dockerfile-parse",
        "methods": {
            "DockerfileParser": {
                "signature": "dockerfile_parse.DockerfileParser(path=None, cache_content=False, env_replace=True, ...) -> DockerfileParser",
                "description": "Parses a Dockerfile. Content reflects user input.",
                "role": "source",
                "tracks": ["return"],
            },
        },
        "rules_using": [],
    },

    "PyToml": {
        "description": (
            "toml parses TOML configuration. toml.load() is a neutral data loader — values "
            "become sources when the config file is user-supplied. tomllib (stdlib, 3.11+) is "
            "the modern replacement."
        ),
        "category": "deserialization",
        "fqns": ["toml", "tomllib"],
        "pip_snippet": "pip install toml  # or use stdlib tomllib on Python 3.11+",
        "methods": {
            "load": {
                "signature": "toml.load(f) -> dict",
                "description": "Parses TOML from a file. Source when file is user-controlled.",
                "role": "source",
                "tracks": ["return"],
            },
            "loads": {
                "signature": "toml.loads(s: str) -> dict",
                "description": "Parses TOML from a string. Source.",
                "role": "source",
                "tracks": ["return"],
            },
        },
        "rules_using": [],
    },

    "PyXmltodict": {
        "description": (
            "xmltodict parses XML into nested dicts (uses expat under the hood). Entity expansion "
            "is disabled by default, but the module's parse() still exposes untrusted XML to the "
            "app. Not a full XXE defense."
        ),
        "category": "deserialization",
        "fqns": ["xmltodict"],
        "pip_snippet": "pip install xmltodict",
        "methods": {
            "parse": {
                "signature": "xmltodict.parse(xml_input, encoding=None, expat=expat, process_namespaces=False, ...) -> dict",
                "description": "Parses XML to dict. Source for user-controlled XML content.",
                "role": "source",
                "tracks": ["return"],
            },
        },
        "rules_using": [],
    },

    "PyWtforms": {
        "description": (
            "WTForms provides form validation for Flask / Django-style apps. Form().validate_on_submit() "
            "is a sanitizer for field-level validation. Still, string field values reach templates / "
            "SQL if fed directly without additional escaping."
        ),
        "category": "web-frameworks",
        "fqns": ["wtforms"],
        "pip_snippet": "pip install wtforms",
        "methods": {
            "Form": {
                "signature": "wtforms.Form(formdata=None, obj=None, prefix='', **kwargs) -> Form",
                "description": "Creates a form.",
                "role": "neutral",
                "tracks": [],
            },
            "Form.validate": {
                "signature": "Form.validate() -> bool",
                "description": "Validates all fields. Sanitizer for shape / type; strings remain tainted.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
        },
        "rules_using": [],
    },

    "PyDjangoFilters": {
        "description": (
            "django-filter builds Django QuerySet filters from query params. FilterSet.qs runs the "
            "filtered query — injection is impossible via the FilterSet, but custom filter methods "
            "that build raw SQL are sinks."
        ),
        "category": "web-frameworks",
        "fqns": ["django_filters"],
        "pip_snippet": "pip install django-filter",
        "methods": {
            "FilterSet": {
                "signature": "django_filters.FilterSet(data=None, queryset=None, request=None, prefix=None)",
                "description": "Builds filtered QuerySet from query params.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
        },
        "rules_using": [],
    },

    "PyPexpect": {
        "description": (
            "pexpect spawns interactive subprocesses with expect/respond patterns. spawn(cmd, ...) "
            "on user-controlled cmd is a command-injection sink, equivalent to subprocess with shell=True."
        ),
        "category": "command-execution",
        "fqns": ["pexpect"],
        "pip_snippet": "pip install pexpect",
        "methods": {
            "spawn": {
                "signature": "pexpect.spawn(command, args=[], timeout=30, ...) -> spawn",
                "description": "Spawns a child process. Command-injection sink on user-controlled command.",
                "role": "sink",
                "tracks": [0],
            },
            "run": {
                "signature": "pexpect.run(command, timeout=30, events=None, extra_args=None, logfile=None, ...) -> bytes",
                "description": "Runs a command and returns output. Sink on user-controlled command.",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": [],
    },

    "PyCffi": {
        "description": (
            "cffi calls C libraries without writing a C extension. FFI.dlopen() loads a shared "
            "library at runtime — code-execution sink on user-controlled path. FFI.cdef parses "
            "C declarations — neutral unless the definitions are user-controlled."
        ),
        "category": "command-execution",
        "fqns": ["cffi", "cffi.FFI"],
        "pip_snippet": "pip install cffi",
        "methods": {
            "FFI": {
                "signature": "cffi.FFI() -> FFI",
                "description": "FFI instance.",
                "role": "neutral",
                "tracks": [],
            },
            "dlopen": {
                "signature": "FFI.dlopen(name, flags=0) -> Library",
                "description": "Loads a shared library. Code-execution sink on user-controlled name.",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": [],
    },
}
