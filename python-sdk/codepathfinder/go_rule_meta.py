# go_rule_meta.py — Companion metadata for go_rule.py SDK classes.
# Read by scripts/generate_sdk_manifest.py to produce sdk-manifest.json.
# Keep in sync with go_rule.py: if you add a class there, add metadata here.

SDK_META: dict = {
    "GoGinContext": {
        "description": (
            "Represents gin.Context, the primary request/response carrier in the Gin HTTP framework. "
            "All user-input accessors (Query, Param, PostForm, etc.) are taint sources. "
            "Output methods (JSON, String, Redirect) are sinks for XSS and open-redirect rules."
        ),
        "category": "web-frameworks",
        "go_mod": "require github.com/gin-gonic/gin v1.9.1",
        "methods": {
            "Query": {
                "signature": "Query(key string) string",
                "description": "Returns URL query parameter value for the given key. Empty string if missing.",
                "role": "source",
                "tracks": ["return"],
            },
            "DefaultQuery": {
                "signature": "DefaultQuery(key, defaultValue string) string",
                "description": "Returns URL query parameter value, or defaultValue if the key is absent.",
                "role": "source",
                "tracks": ["return"],
            },
            "Param": {
                "signature": "Param(key string) string",
                "description": "Returns URL path parameter (e.g. /user/:id). Always non-empty if route matched.",
                "role": "source",
                "tracks": ["return"],
            },
            "PostForm": {
                "signature": "PostForm(key string) string",
                "description": "Returns POST form value for the given key from application/x-www-form-urlencoded body.",
                "role": "source",
                "tracks": ["return"],
            },
            "GetHeader": {
                "signature": "GetHeader(key string) string",
                "description": "Returns HTTP request header value. User-controlled for headers like X-Forwarded-For.",
                "role": "source",
                "tracks": ["return"],
            },
            "ShouldBindJSON": {
                "signature": "ShouldBindJSON(obj any) error",
                "description": "Deserializes JSON request body into obj. obj becomes tainted after binding.",
                "role": "source",
                "tracks": [0],
            },
            "Cookie": {
                "signature": "Cookie(name string) (string, error)",
                "description": "Returns the named cookie value. Cookies are user-controlled.",
                "role": "source",
                "tracks": ["return"],
            },
            "JSON": {
                "signature": "JSON(code int, obj any)",
                "description": "Serializes obj to JSON and writes to response. Sink for reflected XSS if obj contains raw HTML.",
                "role": "sink",
                "tracks": [1],
            },
            "Redirect": {
                "signature": "Redirect(code int, location string)",
                "description": "Redirects to location. Sink for open-redirect if location comes from user input.",
                "role": "sink",
                "tracks": [1],
            },
        },
        "example_rule": """\
from codepathfinder.go_rule import GoGinContext, GoGormDB, GoStrconv
from codepathfinder import flows
from codepathfinder.presets import PropagationPresets
from codepathfinder.go_decorators import go_rule

@go_rule(
    id="GO-GORM-SQLI-001",
    severity="CRITICAL",
    cwe="CWE-89",
    owasp="A03:2021",
    message="User input flows into GORM Raw()/Exec(). Use parameterized queries.",
)
def detect_gorm_sqli():
    return flows(
        from_sources=[
            GoGinContext.method("Query", "Param", "PostForm", "ShouldBindJSON"),
        ],
        to_sinks=[
            GoGormDB.method("Raw", "Exec"),
        ],
        sanitized_by=[
            GoStrconv.method("Atoi", "ParseInt", "ParseFloat"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
""",
        "rules_using": [
            "GO-GORM-SQLI-001",
            "GO-GORM-SQLI-002",
            "GO-SEC-001",
            "GO-SQLI-003",
            "GO-SSRF-001",
            "GO-SEC-002",
            "GO-XSS-001",
            "GO-PATH-001",
            "GO-REDIRECT-001",
        ],
    },
    "GoEchoContext": {
        "description": (
            "Represents echo.Context in the Echo HTTP framework (v4). "
            "Provides typed accessors for all parts of the HTTP request. "
            "All input methods are taint sources."
        ),
        "category": "web-frameworks",
        "go_mod": "require github.com/labstack/echo/v4 v4.11.4",
        "methods": {
            "QueryParam": {
                "signature": "QueryParam(name string) string",
                "description": "Returns URL query parameter value by name.",
                "role": "source",
                "tracks": ["return"],
            },
            "FormValue": {
                "signature": "FormValue(name string) string",
                "description": "Returns POST form value. Reads application/x-www-form-urlencoded or multipart/form-data.",
                "role": "source",
                "tracks": ["return"],
            },
            "Param": {
                "signature": "Param(name string) string",
                "description": "Returns URL path parameter value.",
                "role": "source",
                "tracks": ["return"],
            },
            "Bind": {
                "signature": "Bind(i any) error",
                "description": "Deserializes request body into i based on Content-Type. i becomes tainted.",
                "role": "source",
                "tracks": [0],
            },
            "Redirect": {
                "signature": "Redirect(code int, url string) error",
                "description": "Redirects to url. Sink for open-redirect.",
                "role": "sink",
                "tracks": [1],
            },
        },
        "rules_using": ["GO-SEC-001", "GO-SEC-002", "GO-SSRF-001"],
    },
    "GoFiberCtx": {
        "description": (
            "Represents fiber.Ctx in the Fiber HTTP framework (v2), inspired by Express.js. "
            "Zero-allocation design. All input methods are taint sources."
        ),
        "category": "web-frameworks",
        "go_mod": "require github.com/gofiber/fiber/v2 v2.52.0",
        "methods": {
            "Params": {
                "signature": "Params(key string, defaultValue ...string) string",
                "description": "Returns URL path parameter value.",
                "role": "source",
                "tracks": ["return"],
            },
            "Query": {
                "signature": "Query(key string, defaultValue ...string) string",
                "description": "Returns URL query parameter value.",
                "role": "source",
                "tracks": ["return"],
            },
            "FormValue": {
                "signature": "FormValue(key string, defaultValue ...string) string",
                "description": "Returns POST form value.",
                "role": "source",
                "tracks": ["return"],
            },
            "BodyParser": {
                "signature": "BodyParser(out any) error",
                "description": "Parses request body into out. out becomes tainted.",
                "role": "source",
                "tracks": [0],
            },
            "Redirect": {
                "signature": "Redirect(location string, status ...int) error",
                "description": "Redirects to location. Sink for open-redirect.",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": ["GO-SEC-001", "GO-SEC-002"],
    },
    "GoGormDB": {
        "description": (
            "Represents gorm.DB, the primary database handle in GORM v2. "
            "Raw(), Exec(), and Where() with string arguments are SQL injection sinks "
            "when called with unsanitized user input."
        ),
        "category": "databases",
        "go_mod": "require gorm.io/gorm v1.25.5",
        "methods": {
            "Raw": {
                "signature": "Raw(sql string, values ...any) *DB",
                "description": "Executes raw SQL. The sql string is an injection sink when built with user input.",
                "role": "sink",
                "tracks": [0],
            },
            "Exec": {
                "signature": "Exec(sql string, values ...any) *DB",
                "description": "Executes raw SQL DML. Same risk as Raw().",
                "role": "sink",
                "tracks": [0],
            },
            "Where": {
                "signature": "Where(query any, args ...any) *DB",
                "description": "Adds WHERE clause. Sink when query is a string with user input concatenated.",
                "role": "sink",
                "tracks": [0],
            },
            "Find": {
                "signature": "Find(dest any, conds ...any) *DB",
                "description": "Executes SELECT with optional conditions. Safe when using struct conditions.",
                "role": "neutral",
            },
            "Create": {
                "signature": "Create(value any) *DB",
                "description": "Inserts record. Safe when using struct with parameterized fields.",
                "role": "neutral",
            },
        },
        "example_rule": """\
from codepathfinder.go_rule import GoGinContext, GoGormDB, GoStrconv
from codepathfinder import flows
from codepathfinder.presets import PropagationPresets
from codepathfinder.go_decorators import go_rule

@go_rule(
    id="GO-GORM-SQLI-002",
    severity="HIGH",
    cwe="CWE-89",
    owasp="A03:2021",
    message="String concatenation in GORM query builder. Use ? placeholders.",
)
def detect_gorm_sqli_concat():
    return flows(
        from_sources=[
            GoGinContext.method("Query", "Param", "PostForm"),
        ],
        to_sinks=[
            GoGormDB.method("Where", "Having", "Order"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
""",
        "rules_using": ["GO-GORM-SQLI-001", "GO-GORM-SQLI-002"],
    },
    "GoSqlxDB": {
        "description": (
            "Represents sqlx.DB and sqlx.Tx from the sqlx library, which extends database/sql "
            "with struct scanning. Unsafe query methods (QueryUnsafe, GetUnsafe) and raw "
            "string methods are injection sinks."
        ),
        "category": "databases",
        "go_mod": "require github.com/jmoiron/sqlx v1.3.5",
        "methods": {
            "Query": {
                "signature": "Query(query string, args ...any) (*sql.Rows, error)",
                "description": "Executes raw SQL query. Sink when query string contains user input.",
                "role": "sink",
                "tracks": [0],
            },
            "Exec": {
                "signature": "Exec(query string, args ...any) (sql.Result, error)",
                "description": "Executes raw SQL DML. Sink when query string contains user input.",
                "role": "sink",
                "tracks": [0],
            },
            "Queryx": {
                "signature": "Queryx(query string, args ...any) (*sqlx.Rows, error)",
                "description": "Like Query but returns sqlx.Rows. Same injection risk.",
                "role": "sink",
                "tracks": [0],
            },
            "Get": {
                "signature": "Get(dest any, query string, args ...any) error",
                "description": "Executes query and scans result into dest. query is an injection sink.",
                "role": "sink",
                "tracks": [1],
            },
        },
        "rules_using": ["GO-SQLI-003"],
    },
    "GoSQLDB": {
        "description": (
            "Represents database/sql.DB and database/sql.Tx from the Go standard library. "
            "Query(), Exec(), and Prepare() are SQL injection sinks when the query "
            "string is built from user input instead of using ? placeholders."
        ),
        "category": "stdlib",
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "Query": {
                "signature": "Query(query string, args ...any) (*Rows, error)",
                "description": "Executes parameterized SELECT. Sink when query is built via string concatenation.",
                "role": "sink",
                "tracks": [0],
            },
            "QueryRow": {
                "signature": "QueryRow(query string, args ...any) *Row",
                "description": "Executes parameterized SELECT returning one row. Same injection risk.",
                "role": "sink",
                "tracks": [0],
            },
            "Exec": {
                "signature": "Exec(query string, args ...any) (Result, error)",
                "description": "Executes parameterized DML. Sink when query contains user input.",
                "role": "sink",
                "tracks": [0],
            },
            "Prepare": {
                "signature": "Prepare(query string) (*Stmt, error)",
                "description": "Creates prepared statement. Sink when query string is user-controlled.",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": ["GO-SEC-001"],
    },
    "GoStrconv": {
        "description": (
            "The strconv standard library package. Atoi, ParseInt, ParseFloat, and related "
            "functions serve as sanitizers in SQL injection and path traversal rules — "
            "converting a string to a numeric type eliminates injection risk."
        ),
        "category": "stdlib",
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "Atoi": {
                "signature": "Atoi(s string) (int, error)",
                "description": "Converts string to int. Use as sanitizer: parsed ints cannot inject SQL.",
                "role": "sanitizer",
            },
            "ParseInt": {
                "signature": "ParseInt(s string, base int, bitSize int) (int64, error)",
                "description": "Parses string as integer with given base and bit size. Sanitizes SQL/path injection.",
                "role": "sanitizer",
            },
            "ParseFloat": {
                "signature": "ParseFloat(s string, bitSize int) (float64, error)",
                "description": "Parses string as float. Sanitizes injection via numeric validation.",
                "role": "sanitizer",
            },
            "ParseBool": {
                "signature": "ParseBool(str string) (bool, error)",
                "description": 'Parses "true"/"false" string to bool. Sanitizes by constraining to boolean domain.',
                "role": "sanitizer",
            },
        },
        "rules_using": ["GO-GORM-SQLI-001", "GO-SEC-001"],
    },
    "GoOSExec": {
        "description": (
            "The os/exec standard library package. exec.Command and exec.CommandContext "
            "are command injection sinks when any argument comes from user-controlled input. "
            "Most dangerous with shell=true-equivalent patterns."
        ),
        "category": "stdlib",
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "Command": {
                "signature": "Command(name string, arg ...string) *Cmd",
                "description": "Creates Cmd to run name with args. name and any arg are injection sinks.",
                "role": "sink",
                "tracks": [0],
            },
            "CommandContext": {
                "signature": "CommandContext(ctx context.Context, name string, arg ...string) *Cmd",
                "description": "Like Command but with context for cancellation. Same injection risk.",
                "role": "sink",
                "tracks": [1],
            },
        },
        "rules_using": ["GO-SEC-002"],
    },
    "GoHTTPRequest": {
        "description": (
            "Represents *http.Request from the net/http standard library. "
            "Used in standard http.HandlerFunc handlers. FormValue, URL.Query(), "
            "Header.Get(), and Body are all taint sources."
        ),
        "category": "stdlib",
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "FormValue": {
                "signature": "FormValue(key string) string",
                "description": "Returns the first value for the named POST or query-string field.",
                "role": "source",
                "tracks": ["return"],
            },
            "PostFormValue": {
                "signature": "PostFormValue(key string) string",
                "description": "Returns the first value for the named POST body field only.",
                "role": "source",
                "tracks": ["return"],
            },
            "Header": {
                "signature": "Header.Get(key string) string",
                "description": "Returns the HTTP header value. User-controlled headers like X-Forwarded-For.",
                "role": "source",
                "tracks": ["return"],
            },
            "URL": {
                "signature": "URL.Query().Get(key string) string",
                "description": "URL query string accessor. Equivalent to FormValue for GET params.",
                "role": "source",
                "tracks": ["return"],
            },
        },
        "rules_using": ["GO-SEC-001", "GO-SEC-002", "GO-XSS-001", "GO-SSRF-002"],
    },
    "GoRestyClient": {
        "description": (
            "Represents resty.Client and resty.Request from go-resty/resty v2. "
            "SetURL, Execute, Get, Post etc. are SSRF sinks when the URL comes "
            "from user-controlled input."
        ),
        "category": "http-clients",
        "go_mod": "require github.com/go-resty/resty/v2 v2.11.0",
        "methods": {
            "SetURL": {
                "signature": "SetURL(url string) *Request",
                "description": "Sets the request URL. Sink for SSRF when url is user-controlled.",
                "role": "sink",
                "tracks": [0],
            },
            "Get": {
                "signature": "Get(url string) (*Response, error)",
                "description": "Makes GET request to url. Sink for SSRF.",
                "role": "sink",
                "tracks": [0],
            },
            "Post": {
                "signature": "Post(url string) (*Response, error)",
                "description": "Makes POST request to url. Sink for SSRF.",
                "role": "sink",
                "tracks": [0],
            },
            "Execute": {
                "signature": "Execute(method, url string) (*Response, error)",
                "description": "Makes HTTP request with given method and url. Sink for SSRF.",
                "role": "sink",
                "tracks": [1],
            },
        },
        "rules_using": ["GO-SSRF-001"],
    },
    "GoJWTToken": {
        "description": (
            "Represents jwt.Token from github.com/golang-jwt/jwt v5. "
            "The Valid field and Parse function are critical — rules detect patterns "
            "where signature verification is skipped."
        ),
        "category": "auth-config",
        "go_mod": "require github.com/golang-jwt/jwt/v5 v5.2.0",
        "methods": {
            "ParseWithClaims": {
                "signature": "ParseWithClaims(tokenString string, claims Claims, keyFunc Keyfunc, options ...ParserOption) (*Token, error)",
                "description": "Parses and validates JWT. keyFunc returning nil skips signature verification.",
                "role": "sink",
                "tracks": [0],
            },
            "Valid": {
                "signature": "Valid bool (field)",
                "description": "True if the token was validated. Accessing claims without checking Valid is a finding.",
                "role": "neutral",
            },
        },
        "rules_using": ["GO-JWT-002"],
    },
    # ── stdlib already in go_rule.py but not yet in meta ──────────────────────
    "GoHTTPResponseWriter": {
        "description": "Represents net/http.ResponseWriter. Write() and WriteString() are XSS sinks when writing unsanitized user input into the HTTP response body.",
        "category": "stdlib",
        "fqns": ["net/http.ResponseWriter"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "Write": {
                "signature": "Write(b []byte) (int, error)",
                "description": "Writes raw bytes to response. XSS sink when b contains user input.",
                "role": "sink",
                "tracks": [0],
            },
            "WriteHeader": {
                "signature": "WriteHeader(statusCode int)",
                "description": "Sets HTTP status code. Not a taint sink.",
                "role": "neutral",
            },
        },
        "rules_using": ["GO-XSS-001"],
    },
    "GoHTTPClient": {
        "description": "Represents net/http.Client. Do(), Get(), Post() are SSRF sinks when the URL comes from user input.",
        "category": "stdlib",
        "fqns": ["net/http.Client"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "Get": {
                "signature": "Get(url string) (*Response, error)",
                "description": "Makes GET request. SSRF sink when url is user-controlled.",
                "role": "sink",
                "tracks": [0],
            },
            "Post": {
                "signature": "Post(url, contentType string, body io.Reader) (*Response, error)",
                "description": "Makes POST request. SSRF sink when url is user-controlled.",
                "role": "sink",
                "tracks": [0],
            },
            "Do": {
                "signature": "Do(req *Request) (*Response, error)",
                "description": "Executes arbitrary HTTP request. SSRF sink.",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": ["GO-SSRF-002"],
    },
    "GoOS": {
        "description": "The os standard library package. Getenv() is a source of environment variable data. Open(), Create(), Remove() are file operation sinks for path traversal.",
        "category": "stdlib",
        "fqns": ["os"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "Getenv": {
                "signature": "Getenv(key string) string",
                "description": "Returns environment variable value. Source of external data.",
                "role": "source",
                "tracks": ["return"],
            },
            "Open": {
                "signature": "Open(name string) (*File, error)",
                "description": "Opens file for reading. Path traversal sink when name is user-controlled.",
                "role": "sink",
                "tracks": [0],
            },
            "Create": {
                "signature": "Create(name string) (*File, error)",
                "description": "Creates file. Path traversal sink when name is user-controlled.",
                "role": "sink",
                "tracks": [0],
            },
            "Remove": {
                "signature": "Remove(name string) error",
                "description": "Removes file. Dangerous sink when name is user-controlled.",
                "role": "sink",
                "tracks": [0],
            },
            "ReadFile": {
                "signature": "ReadFile(name string) ([]byte, error)",
                "description": "Reads entire file. Path traversal sink.",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": ["GO-PATH-001"],
    },
    "GoFilepath": {
        "description": "The path/filepath standard library package. Join(), Abs(), Clean() are used as sanitizers in path traversal rules when combined with containment checks.",
        "category": "stdlib",
        "fqns": ["path/filepath"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "Join": {
                "signature": "Join(elem ...string) string",
                "description": "Joins path elements. Sanitizer when followed by a prefix containment check.",
                "role": "sanitizer",
            },
            "Abs": {
                "signature": "Abs(path string) (string, error)",
                "description": "Returns absolute path. Sanitizer when result is checked against allowed root.",
                "role": "sanitizer",
            },
            "Clean": {
                "signature": "Clean(path string) string",
                "description": "Lexically cleans path. Partial sanitizer — still needs containment check.",
                "role": "sanitizer",
            },
            "Base": {
                "signature": "Base(path string) string",
                "description": "Returns last element of path. Strips directory traversal sequences.",
                "role": "sanitizer",
            },
        },
        "rules_using": ["GO-PATH-001"],
    },
    "GoFmt": {
        "description": "The fmt standard library package. Sprintf, Fprintf, Sscanf are sources of formatted string data. Fprintf to http.ResponseWriter is an XSS sink.",
        "category": "stdlib",
        "fqns": ["fmt"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "Sprintf": {
                "signature": "Sprintf(format string, a ...any) string",
                "description": "Formats string. Propagates taint from arguments into the return value.",
                "role": "neutral",
            },
            "Fprintf": {
                "signature": "Fprintf(w io.Writer, format string, a ...any) (n int, err error)",
                "description": "Writes to w. XSS sink when w is http.ResponseWriter and a contains user input.",
                "role": "sink",
                "tracks": [1],
            },
            "Sscanf": {
                "signature": "Sscanf(str string, format string, a ...any) (n int, err error)",
                "description": "Parses str. a arguments become tainted with str contents.",
                "role": "source",
            },
        },
        "rules_using": [],
    },
    "GoTemplate": {
        "description": "Represents html/template.Template and text/template.Template. Execute() and ExecuteTemplate() are XSS sinks when data contains unsanitized user input passed to text/template (not html/template).",
        "category": "stdlib",
        "fqns": ["html/template.Template", "text/template.Template"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "Execute": {
                "signature": "Execute(wr io.Writer, data any) error",
                "description": "Renders template with data. XSS sink for text/template when data is user-controlled.",
                "role": "sink",
                "tracks": [1],
            },
            "ExecuteTemplate": {
                "signature": "ExecuteTemplate(wr io.Writer, name string, data any) error",
                "description": "Renders named template. Same XSS risk as Execute.",
                "role": "sink",
                "tracks": [2],
            },
            "Parse": {
                "signature": "Parse(text string) (*Template, error)",
                "description": "Parses template text. Server-side template injection if text is user-controlled.",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": ["GO-XSS-003"],
    },
    "GoCrypto": {
        "description": "Weak cryptographic algorithms: crypto/md5, crypto/sha1, crypto/des, crypto/rc4. All New() and Sum() calls are findings — these algorithms are cryptographically broken.",
        "category": "stdlib",
        "fqns": ["crypto/md5", "crypto/sha1", "crypto/des", "crypto/rc4"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "New": {
                "signature": "New() hash.Hash",
                "description": "Creates new hash instance using the weak algorithm. Always a finding.",
                "role": "sink",
            },
            "Sum": {
                "signature": "Sum(data []byte) [N]byte",
                "description": "Computes weak hash. Always a finding.",
                "role": "sink",
            },
        },
        "rules_using": [
            "GO-CRYPTO-001",
            "GO-CRYPTO-002",
            "GO-CRYPTO-003",
            "GO-CRYPTO-004",
            "GO-CRYPTO-005",
        ],
    },
    "GoContext": {
        "description": "Represents context.Context. Value() can propagate tainted data stored by upstream handlers — treat returned values as taint sources in inter-procedural analysis.",
        "category": "stdlib",
        "fqns": ["context.Context"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "Value": {
                "signature": "Value(key any) any",
                "description": "Returns value associated with key. Source of taint when key is a request-scoped user-data key.",
                "role": "source",
                "tracks": ["return"],
            },
            "WithValue": {
                "signature": "WithValue(parent Context, key, val any) Context",
                "description": "Returns context carrying val. Propagates taint from val.",
                "role": "neutral",
            },
        },
        "rules_using": [],
    },
    # ── 40+ additional stdlib entries for UI scale testing ────────────────────
    "GoBufioReader": {
        "description": "bufio.Reader wraps an io.Reader with buffering. ReadString() and ReadLine() are sources when the underlying reader is an HTTP request body or stdin.",
        "category": "stdlib",
        "fqns": ["bufio.Reader"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "ReadString": {
                "signature": "ReadString(delim byte) (string, error)",
                "description": "Reads until delimiter. Source when wrapping user-controlled input.",
                "role": "source",
                "tracks": ["return"],
            },
            "ReadLine": {
                "signature": "ReadLine() (line []byte, isPrefix bool, err error)",
                "description": "Reads one line. Source when wrapping HTTP body or stdin.",
                "role": "source",
                "tracks": ["return"],
            },
            "ReadBytes": {
                "signature": "ReadBytes(delim byte) ([]byte, error)",
                "description": "Reads until delimiter. Source of tainted bytes.",
                "role": "source",
                "tracks": ["return"],
            },
        },
        "rules_using": [],
    },
    "GoBufioScanner": {
        "description": "bufio.Scanner reads tokens line-by-line. Text() and Bytes() are sources when the scanner wraps user-controlled input (stdin, HTTP body).",
        "category": "stdlib",
        "fqns": ["bufio.Scanner"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "Text": {
                "signature": "Text() string",
                "description": "Returns current token as string. Source when scanning user input.",
                "role": "source",
                "tracks": ["return"],
            },
            "Bytes": {
                "signature": "Bytes() []byte",
                "description": "Returns current token as bytes. Source when scanning user input.",
                "role": "source",
                "tracks": ["return"],
            },
        },
        "rules_using": [],
    },
    "GoIOReader": {
        "description": "io.Reader interface. ReadAll() from io package returns the full content of a reader — source of taint when the reader wraps HTTP request body.",
        "category": "stdlib",
        "fqns": ["io"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "ReadAll": {
                "signature": "ReadAll(r Reader) ([]byte, error)",
                "description": "Reads all bytes from r. Source when r is http.Request.Body.",
                "role": "source",
                "tracks": ["return"],
            },
            "Copy": {
                "signature": "Copy(dst Writer, src Reader) (int64, error)",
                "description": "Copies src to dst. Propagates taint from src to dst.",
                "role": "neutral",
            },
            "Pipe": {
                "signature": "Pipe() (*PipeReader, *PipeWriter)",
                "description": "Creates synchronized pipe. Propagates taint through the connection.",
                "role": "neutral",
            },
        },
        "rules_using": [],
    },
    "GoNetURL": {
        "description": "net/url package. Parse() returns a *url.URL from a string — source of taint when parsing user-supplied URLs. Used in SSRF detection for URL validation.",
        "category": "stdlib",
        "fqns": ["net/url"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "Parse": {
                "signature": "Parse(rawURL string) (*URL, error)",
                "description": "Parses raw URL. Sanitizer when result host is validated against allowlist.",
                "role": "sanitizer",
            },
            "QueryUnescape": {
                "signature": "QueryUnescape(s string) (string, error)",
                "description": "Decodes percent-encoded string. Returns decoded tainted data.",
                "role": "neutral",
            },
            "PathEscape": {
                "signature": "PathEscape(s string) string",
                "description": "Escapes string for use in URL path segment. Sanitizes path injection.",
                "role": "sanitizer",
            },
            "QueryEscape": {
                "signature": "QueryEscape(s string) string",
                "description": "Escapes string for use in URL query. Sanitizes injection via encoding.",
                "role": "sanitizer",
            },
        },
        "rules_using": [],
    },
    "GoNetDial": {
        "description": "net.Dial and net.DialTCP create network connections. Dial() is an SSRF sink when the address is user-controlled.",
        "category": "stdlib",
        "fqns": ["net"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "Dial": {
                "signature": "Dial(network, address string) (Conn, error)",
                "description": "Creates network connection to address. SSRF sink when address is user-controlled.",
                "role": "sink",
                "tracks": [1],
            },
            "DialTCP": {
                "signature": "DialTCP(network string, laddr, raddr *TCPAddr) (*TCPConn, error)",
                "description": "Creates TCP connection. SSRF sink when raddr is user-controlled.",
                "role": "sink",
                "tracks": [2],
            },
            "LookupHost": {
                "signature": "LookupHost(host string) ([]string, error)",
                "description": "DNS lookup. SSRF vector when host is user-controlled.",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": [],
    },
    "GoNetTLS": {
        "description": "crypto/tls package. Config.InsecureSkipVerify = true disables certificate verification — a finding for all production code.",
        "category": "stdlib",
        "fqns": ["crypto/tls"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "Dial": {
                "signature": "Dial(network, addr string, config *Config) (*Conn, error)",
                "description": "Creates TLS connection. Finding when config.InsecureSkipVerify is true.",
                "role": "sink",
                "tracks": [2],
            },
        },
        "rules_using": [],
    },
    "GoEncodingBase64": {
        "description": "encoding/base64 package. DecodeString() decodes user input — the result is still tainted and must be sanitized before use in sinks.",
        "category": "stdlib",
        "fqns": ["encoding/base64"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "DecodeString": {
                "signature": "DecodeString(s string) ([]byte, error)",
                "description": "Decodes base64. Output is tainted if input is tainted — not a sanitizer.",
                "role": "neutral",
            },
            "EncodeToString": {
                "signature": "EncodeToString(src []byte) string",
                "description": "Encodes to base64. Does not sanitize — taint propagates.",
                "role": "neutral",
            },
        },
        "rules_using": [],
    },
    "GoEncodingHex": {
        "description": "encoding/hex package. DecodeString() converts hex to bytes — does not sanitize taint. EncodeToString() may be used as a sanitizer in specific contexts.",
        "category": "stdlib",
        "fqns": ["encoding/hex"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "DecodeString": {
                "signature": "DecodeString(s string) ([]byte, error)",
                "description": "Decodes hex string to bytes. Taint propagates through.",
                "role": "neutral",
            },
            "EncodeToString": {
                "signature": "EncodeToString(src []byte) string",
                "description": "Encodes bytes to hex. Safe for SQL/command contexts — acts as sanitizer.",
                "role": "sanitizer",
            },
        },
        "rules_using": [],
    },
    "GoEncodingJSON": {
        "description": "encoding/json package. Unmarshal and Decoder.Decode() are sources of tainted data from JSON input. Marshal() propagates taint to output.",
        "category": "stdlib",
        "fqns": ["encoding/json"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "Unmarshal": {
                "signature": "Unmarshal(data []byte, v any) error",
                "description": "Decodes JSON into v. v becomes tainted when data comes from user input.",
                "role": "source",
                "tracks": [1],
            },
            "Marshal": {
                "signature": "Marshal(v any) ([]byte, error)",
                "description": "Encodes v to JSON. Propagates taint from v to output bytes.",
                "role": "neutral",
            },
        },
        "rules_using": [],
    },
    "GoEncodingXML": {
        "description": "encoding/xml package. Unmarshal and Decoder.Decode() are sources. Can also be an XXE sink if xml.Decoder is used without disabling external entity processing.",
        "category": "stdlib",
        "fqns": ["encoding/xml"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "Unmarshal": {
                "signature": "Unmarshal(data []byte, v any) error",
                "description": "Decodes XML into v. v becomes tainted. Potential XXE if data contains external entities.",
                "role": "source",
                "tracks": [1],
            },
            "NewDecoder": {
                "signature": "NewDecoder(r io.Reader) *Decoder",
                "description": "Creates XML decoder. XXE risk when r is user-controlled and entity expansion not limited.",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": [],
    },
    "GoEncodingCSV": {
        "description": "encoding/csv package. Reader.Read() and Reader.ReadAll() return user-controlled CSV data as string slices — treat as taint sources.",
        "category": "stdlib",
        "fqns": ["encoding/csv"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "Read": {
                "signature": "Read() ([]string, error)",
                "description": "Reads one CSV record. Source of tainted strings when reading user-uploaded CSV.",
                "role": "source",
                "tracks": ["return"],
            },
            "ReadAll": {
                "signature": "ReadAll() ([][]string, error)",
                "description": "Reads all CSV records. Source of tainted string slices.",
                "role": "source",
                "tracks": ["return"],
            },
        },
        "rules_using": [],
    },
    "GoEncodingBinary": {
        "description": "encoding/binary package. Read() deserializes binary data from a reader — source of taint when the reader is network or user input.",
        "category": "stdlib",
        "fqns": ["encoding/binary"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "Read": {
                "signature": "Read(r io.Reader, order ByteOrder, data any) error",
                "description": "Reads binary data into data. data becomes tainted when r is user-controlled.",
                "role": "source",
                "tracks": [2],
            },
        },
        "rules_using": [],
    },
    "GoEncodingGob": {
        "description": "encoding/gob package. Decoder.Decode() deserializes arbitrary Go types — unsafe deserialization sink when decoding untrusted data.",
        "category": "stdlib",
        "fqns": ["encoding/gob"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "Decode": {
                "signature": "Decode(e any) error",
                "description": "Deserializes gob-encoded data. Unsafe deserialization sink when decoding user-controlled data.",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": [],
    },
    "GoMimeMultipart": {
        "description": "mime/multipart package. Reader.ReadForm() parses multipart form data including file uploads — source of user-controlled filenames and content.",
        "category": "stdlib",
        "fqns": ["mime/multipart"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "ReadForm": {
                "signature": "ReadForm(maxMemory int64) (*Form, error)",
                "description": "Parses entire multipart form including uploads. Source of user-controlled filenames.",
                "role": "source",
                "tracks": ["return"],
            },
            "NextPart": {
                "signature": "NextPart() (*Part, error)",
                "description": "Returns next form part. FileName() on the result is user-controlled.",
                "role": "source",
                "tracks": ["return"],
            },
        },
        "rules_using": [],
    },
    "GoHTTPMux": {
        "description": "net/http.ServeMux is the HTTP request multiplexer. Handle() and HandleFunc() register handlers — not typically a security sink but relevant for routing analysis.",
        "category": "stdlib",
        "fqns": ["net/http.ServeMux"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "Handle": {
                "signature": "Handle(pattern string, handler Handler)",
                "description": "Registers handler for pattern. Not a security sink.",
                "role": "neutral",
            },
            "HandleFunc": {
                "signature": "HandleFunc(pattern string, handler func(ResponseWriter, *Request))",
                "description": "Registers handler function. Not a security sink.",
                "role": "neutral",
            },
        },
        "rules_using": [],
    },
    "GoHTTPServer": {
        "description": "net/http.Server. ListenAndServe() without TLS is a finding in server configurations that should enforce HTTPS.",
        "category": "stdlib",
        "fqns": ["net/http.Server"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "ListenAndServe": {
                "signature": "ListenAndServe() error",
                "description": "Starts HTTP server without TLS. Finding when used in production without HTTPS redirect.",
                "role": "sink",
            },
            "ListenAndServeTLS": {
                "signature": "ListenAndServeTLS(certFile, keyFile string) error",
                "description": "Starts HTTPS server. Safe — preferred over ListenAndServe.",
                "role": "neutral",
            },
        },
        "rules_using": ["GO-NET-001"],
    },
    "GoHTTPCookie": {
        "description": "net/http.Cookie struct. Missing Secure, HttpOnly, or SameSite flags are security findings for session cookies.",
        "category": "stdlib",
        "fqns": ["net/http.Cookie"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "SetCookie": {
                "signature": "SetCookie(w ResponseWriter, cookie *Cookie)",
                "description": "Sets HTTP cookie. Finding when cookie.Secure or cookie.HttpOnly is false for session cookies.",
                "role": "sink",
                "tracks": [1],
            },
        },
        "rules_using": [],
    },
    "GoNetSMTP": {
        "description": "net/smtp package. SendMail() and SMTP.Mail() are email injection sinks when headers or body are built from user input without sanitization.",
        "category": "stdlib",
        "fqns": ["net/smtp"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "SendMail": {
                "signature": "SendMail(addr string, a Auth, from string, to []string, msg []byte) error",
                "description": "Sends email. Header injection sink when from/to/msg contain user input.",
                "role": "sink",
                "tracks": [2],
            },
        },
        "rules_using": [],
    },
    "GoLog": {
        "description": "log standard library package. Printf, Println, and Fatal variants may log sensitive user input — a finding for privacy/compliance rules.",
        "category": "stdlib",
        "fqns": ["log"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "Printf": {
                "signature": "Printf(format string, v ...any)",
                "description": "Logs formatted message. Log injection sink when v contains user input with newlines.",
                "role": "sink",
                "tracks": [0],
            },
            "Println": {
                "signature": "Println(v ...any)",
                "description": "Logs values. Potential log injection.",
                "role": "sink",
                "tracks": [0],
            },
            "Fatal": {
                "signature": "Fatal(v ...any)",
                "description": "Logs and calls os.Exit(1). Log injection sink.",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": [],
    },
    "GoLogSlog": {
        "description": "log/slog package (Go 1.21+). Structured logging — Info, Warn, Error are log injection sinks when message or attributes contain unsanitized user input.",
        "category": "stdlib",
        "fqns": ["log/slog"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "Info": {
                "signature": "Info(msg string, args ...any)",
                "description": "Logs at INFO level. Log injection sink when msg or args contain user input.",
                "role": "sink",
                "tracks": [0],
            },
            "Warn": {
                "signature": "Warn(msg string, args ...any)",
                "description": "Logs at WARN level. Log injection sink.",
                "role": "sink",
                "tracks": [0],
            },
            "Error": {
                "signature": "Error(msg string, args ...any)",
                "description": "Logs at ERROR level. Log injection sink.",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": [],
    },
    "GoStrings": {
        "description": "strings package. Contains(), HasPrefix(), ReplaceAll() are used as partial sanitizers. Builder is used to construct tainted strings.",
        "category": "stdlib",
        "fqns": ["strings"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "Contains": {
                "signature": "Contains(s, substr string) bool",
                "description": "Checks if s contains substr. Used as a partial path containment sanitizer.",
                "role": "sanitizer",
            },
            "HasPrefix": {
                "signature": "HasPrefix(s, prefix string) bool",
                "description": "Checks string prefix. Partial sanitizer for path traversal when checking allowed root.",
                "role": "sanitizer",
            },
            "ReplaceAll": {
                "signature": "ReplaceAll(s, old, new string) string",
                "description": "Replaces all occurrences. Taint propagates — not a sanitizer by itself.",
                "role": "neutral",
            },
            "TrimSpace": {
                "signature": "TrimSpace(s string) string",
                "description": "Trims whitespace. Taint propagates.",
                "role": "neutral",
            },
        },
        "rules_using": [],
    },
    "GoRegexp": {
        "description": "regexp package. FindString() and FindAllString() return tainted matches. MustCompile() with user-controlled pattern is a ReDoS risk.",
        "category": "stdlib",
        "fqns": ["regexp"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "Compile": {
                "signature": "Compile(expr string) (*Regexp, error)",
                "description": "Compiles regex. ReDoS risk when expr is user-controlled.",
                "role": "sink",
                "tracks": [0],
            },
            "MustCompile": {
                "signature": "MustCompile(str string) *Regexp",
                "description": "Compiles regex, panics on error. ReDoS risk when str is user-controlled.",
                "role": "sink",
                "tracks": [0],
            },
            "FindString": {
                "signature": "FindString(s string) string",
                "description": "Returns leftmost match. Source of tainted string from user input.",
                "role": "neutral",
            },
        },
        "rules_using": [],
    },
    "GoMathRand": {
        "description": "math/rand package. Intn(), Float64() and related functions use a deterministic PRNG — a finding when used for cryptographic purposes (tokens, session IDs).",
        "category": "stdlib",
        "fqns": ["math/rand"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "Intn": {
                "signature": "Intn(n int) int",
                "description": "Returns pseudo-random int. Finding when used to generate security tokens.",
                "role": "sink",
            },
            "Read": {
                "signature": "Read(p []byte) (n int, err error)",
                "description": "Fills p with pseudo-random bytes. Finding when used as cryptographic randomness.",
                "role": "sink",
            },
        },
        "rules_using": [],
    },
    "GoCryptoRand": {
        "description": "crypto/rand package. The Reader is the cryptographically secure random source — use this instead of math/rand for tokens and session IDs.",
        "category": "stdlib",
        "fqns": ["crypto/rand"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "Read": {
                "signature": "Read(b []byte) (n int, err error)",
                "description": "Fills b with cryptographically secure random bytes. Preferred over math/rand.Read.",
                "role": "neutral",
            },
            "Int": {
                "signature": "Int(rand io.Reader, max *big.Int) (*big.Int, error)",
                "description": "Returns cryptographically secure random int. Safe for security purposes.",
                "role": "neutral",
            },
        },
        "rules_using": [],
    },
    "GoSync": {
        "description": "sync package. Mutex, RWMutex, Once — not security sinks but relevant for race condition detection rules.",
        "category": "stdlib",
        "fqns": ["sync"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "Lock": {
                "signature": "Lock()",
                "description": "Acquires mutex. Missing unlock is a resource leak finding.",
                "role": "neutral",
            },
            "Unlock": {
                "signature": "Unlock()",
                "description": "Releases mutex. Must be called, typically via defer.",
                "role": "neutral",
            },
        },
        "rules_using": [],
    },
    "GoSyncMap": {
        "description": "sync.Map provides a concurrent map. Load() and Store() are relevant for data flow tracking in concurrent handlers where shared state is modified.",
        "category": "stdlib",
        "fqns": ["sync.Map"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "Load": {
                "signature": "Load(key any) (value any, ok bool)",
                "description": "Loads value from map. Source of taint when map stores user-controlled data.",
                "role": "source",
                "tracks": ["return"],
            },
            "Store": {
                "signature": "Store(key, value any)",
                "description": "Stores value in map. Propagates taint from value.",
                "role": "neutral",
            },
        },
        "rules_using": [],
    },
    "GoReflect": {
        "description": "reflect package. reflect.ValueOf() and reflect.New() with user-controlled type strings enable dynamic code execution — a finding for unsafe reflection rules.",
        "category": "stdlib",
        "fqns": ["reflect"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "ValueOf": {
                "signature": "ValueOf(i any) Value",
                "description": "Returns Value wrapping i. Taint propagates through reflect operations.",
                "role": "neutral",
            },
            "New": {
                "signature": "New(typ Type) Value",
                "description": "Creates new zero value of type. Unsafe when type is derived from user input.",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": [],
    },
    "GoRuntime": {
        "description": "runtime package. SetFinalizer(), GOMAXPROCS() — not typical security sinks but relevant for resource exhaustion rules.",
        "category": "stdlib",
        "fqns": ["runtime"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "GOMAXPROCS": {
                "signature": "GOMAXPROCS(n int) int",
                "description": "Sets max OS threads. DoS risk when n is derived from user input without bounds check.",
                "role": "sink",
                "tracks": [0],
            },
            "Stack": {
                "signature": "Stack(buf []byte, all bool) int",
                "description": "Writes goroutine stack trace. Information disclosure if written to user-visible output.",
                "role": "source",
            },
        },
        "rules_using": [],
    },
    "GoOSUser": {
        "description": "os/user package. Lookup() and LookupId() resolve usernames — source of OS-level user data. Relevant for privilege escalation analysis.",
        "category": "stdlib",
        "fqns": ["os/user"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "Lookup": {
                "signature": "Lookup(username string) (*User, error)",
                "description": "Looks up user by name. SSRF-like sink if username is user-controlled.",
                "role": "sink",
                "tracks": [0],
            },
            "Current": {
                "signature": "Current() (*User, error)",
                "description": "Returns current OS user. Source of sensitive system information.",
                "role": "source",
                "tracks": ["return"],
            },
        },
        "rules_using": [],
    },
    "GoSyscall": {
        "description": "syscall package. Exec(), RawSyscall(), and socket operations are low-level command and network injection sinks.",
        "category": "stdlib",
        "fqns": ["syscall"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "Exec": {
                "signature": "Exec(argv0 string, argv []string, envv []string) error",
                "description": "Executes program directly. Command injection sink when argv is user-controlled.",
                "role": "sink",
                "tracks": [0],
            },
            "Getenv": {
                "signature": "Getenv(key string) (value string, found bool)",
                "description": "Gets environment variable. Source of external data.",
                "role": "source",
                "tracks": ["return"],
            },
        },
        "rules_using": [],
    },
    "GoIOFS": {
        "description": "io/fs package (Go 1.16+). FS interface and ReadFile() operate on filesystem abstractions — path traversal sinks when path is user-controlled.",
        "category": "stdlib",
        "fqns": ["io/fs"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "ReadFile": {
                "signature": "ReadFile(fsys FS, name string) ([]byte, error)",
                "description": "Reads file from FS. Path traversal sink when name is user-controlled.",
                "role": "sink",
                "tracks": [1],
            },
            "Stat": {
                "signature": "Stat(fsys FS, name string) (FileInfo, error)",
                "description": "Stats file. Path traversal sink when name is user-controlled.",
                "role": "sink",
                "tracks": [1],
            },
        },
        "rules_using": [],
    },
    "GoHTMLTemplate": {
        "description": "html/template package — the safe version of text/template. Auto-escapes context-appropriately. HTML(), JS(), URL() types are escape bypasses when used with user input.",
        "category": "stdlib",
        "fqns": ["html/template"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "HTML": {
                "signature": "HTML(string)",
                "description": "Marks string as safe HTML — bypasses auto-escaping. XSS sink when value is user-controlled.",
                "role": "sink",
                "tracks": [0],
            },
            "JS": {
                "signature": "JS(string)",
                "description": "Marks string as safe JavaScript — bypasses auto-escaping. XSS sink.",
                "role": "sink",
                "tracks": [0],
            },
            "URL": {
                "signature": "URL(string)",
                "description": "Marks string as safe URL — bypasses sanitization. Open redirect sink.",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": ["GO-XSS-002"],
    },
    "GoNetHTTP": {
        "description": "Package-level net/http functions: Get(), Post(), Head(). SSRF sinks when the URL argument is derived from user input.",
        "category": "stdlib",
        "fqns": ["net/http"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "Get": {
                "signature": "Get(url string) (*Response, error)",
                "description": "Package-level HTTP GET. SSRF sink when url is user-controlled.",
                "role": "sink",
                "tracks": [0],
            },
            "Post": {
                "signature": "Post(url, contentType string, body io.Reader) (*Response, error)",
                "description": "Package-level HTTP POST. SSRF sink when url is user-controlled.",
                "role": "sink",
                "tracks": [0],
            },
            "Head": {
                "signature": "Head(url string) (*Response, error)",
                "description": "Package-level HTTP HEAD. SSRF sink.",
                "role": "sink",
                "tracks": [0],
            },
            "Redirect": {
                "signature": "Redirect(w ResponseWriter, r *Request, url string, code int)",
                "description": "Sends redirect response. Open redirect sink when url is user-controlled.",
                "role": "sink",
                "tracks": [2],
            },
        },
        "rules_using": ["GO-SSRF-002", "GO-REDIRECT-001"],
    },
    "GoArchiveTar": {
        "description": "archive/tar package. Reader.Next() returns headers with user-controlled filenames — Zip Slip path traversal sink when extracting to filesystem.",
        "category": "stdlib",
        "fqns": ["archive/tar"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "Next": {
                "signature": "Next() (*Header, error)",
                "description": "Advances to next entry. Header.Name is user-controlled — Zip Slip path traversal sink.",
                "role": "source",
                "tracks": ["return"],
            },
        },
        "rules_using": [],
    },
    "GoArchiveZip": {
        "description": "archive/zip package. OpenReader() and File[].Name are sources of user-controlled filenames — Zip Slip path traversal when extracting.",
        "category": "stdlib",
        "fqns": ["archive/zip"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "OpenReader": {
                "signature": "OpenReader(name string) (*ReadCloser, error)",
                "description": "Opens zip file for reading. File.Name fields are user-controlled — Zip Slip source.",
                "role": "source",
            },
        },
        "rules_using": [],
    },
    "GoDatabaseSQL": {
        "description": "Alias reference: database/sql.Stmt. Prepared statement execution methods — safe when using ? placeholders, sink when mixing with string concatenation.",
        "category": "stdlib",
        "fqns": ["database/sql.Stmt"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "Exec": {
                "signature": "Exec(args ...any) (Result, error)",
                "description": "Executes prepared statement. Safe with parameterized args.",
                "role": "neutral",
            },
            "Query": {
                "signature": "Query(args ...any) (*Rows, error)",
                "description": "Executes parameterized query. Safe with ? placeholders.",
                "role": "neutral",
            },
            "QueryRow": {
                "signature": "QueryRow(args ...any) *Row",
                "description": "Executes parameterized single-row query. Safe with ? placeholders.",
                "role": "neutral",
            },
        },
        "rules_using": [],
    },
    "GoTime": {
        "description": "time package. time.Parse() with user-controlled layout strings is a denial-of-service risk (algorithmic complexity). Not a typical injection sink.",
        "category": "stdlib",
        "fqns": ["time"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "Parse": {
                "signature": "Parse(layout, value string) (Time, error)",
                "description": "Parses time string. Not an injection sink but DoS risk with user-controlled layout.",
                "role": "neutral",
            },
            "After": {
                "signature": "After(d Duration) <-chan Time",
                "description": "Returns channel that fires after d. DoS risk when d is user-controlled without bounds.",
                "role": "neutral",
            },
        },
        "rules_using": [],
    },
    "GoPlugin": {
        "description": "plugin package. Open() loads a shared library — code execution sink when the plugin path is user-controlled.",
        "category": "stdlib",
        "fqns": ["plugin"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "Open": {
                "signature": "Open(path string) (*Plugin, error)",
                "description": "Loads a shared object plugin. Code execution sink when path is user-controlled.",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": [],
    },
    "GoCryptoHMAC": {
        "description": "crypto/hmac package. New() creates HMAC with a key. Equal() provides constant-time comparison. Using == instead of Equal() for MAC verification is a timing attack.",
        "category": "stdlib",
        "fqns": ["crypto/hmac"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "New": {
                "signature": "New(h func() hash.Hash, key []byte) hash.Hash",
                "description": "Creates new HMAC. Safe when using strong hash (sha256, sha512).",
                "role": "neutral",
            },
            "Equal": {
                "signature": "Equal(mac1, mac2 []byte) bool",
                "description": "Constant-time comparison. Use this instead of bytes.Equal for MAC verification.",
                "role": "sanitizer",
            },
        },
        "rules_using": [],
    },
    "GoCryptoAES": {
        "description": "crypto/aes package. NewCipher() with a weak mode (ECB, CBC without IV) is a cryptographic weakness finding.",
        "category": "stdlib",
        "fqns": ["crypto/aes"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "NewCipher": {
                "signature": "NewCipher(key []byte) (cipher.Block, error)",
                "description": "Creates AES block cipher. Finding when used in ECB mode (no IV).",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": [],
    },
    "GoCipherGCM": {
        "description": "cipher package. NewGCMWithNonceSize() and AEAD.Seal() — finding when nonce is reused or predictable.",
        "category": "stdlib",
        "fqns": ["crypto/cipher"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "NewGCM": {
                "signature": "NewGCM(cipher Block) (AEAD, error)",
                "description": "Creates GCM mode cipher. Finding when nonce is not cryptographically random.",
                "role": "neutral",
            },
            "Seal": {
                "signature": "Seal(dst, nonce, plaintext, additionalData []byte) []byte",
                "description": "Encrypts and authenticates. Finding when nonce is reused.",
                "role": "sink",
                "tracks": [1],
            },
        },
        "rules_using": [],
    },
    "GoX509": {
        "description": "crypto/x509 package. Certificate.Verify() is the TLS chain validation entry point. Skipping verification or using empty VerifyOptions is a finding.",
        "category": "stdlib",
        "fqns": ["crypto/x509"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "Verify": {
                "signature": "Verify(opts VerifyOptions) ([][]*Certificate, error)",
                "description": "Verifies certificate chain. Finding when opts is empty (no root CA check).",
                "role": "sink",
                "tracks": [0],
            },
            "ParseCertificate": {
                "signature": "ParseCertificate(asn1Data []byte) (*Certificate, error)",
                "description": "Parses DER-encoded certificate. Source of cert data from network input.",
                "role": "source",
            },
        },
        "rules_using": [],
    },
    "GoGobDecoder": {
        "description": "encoding/gob.Decoder. Decode() deserializes arbitrary Go values — unsafe deserialization when decoding user-supplied bytes.",
        "category": "stdlib",
        "fqns": ["encoding/gob.Decoder"],
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "Decode": {
                "signature": "Decode(e any) error",
                "description": "Deserializes next gob value into e. Unsafe deserialization sink.",
                "role": "sink",
                "tracks": [0],
            },
            "DecodeValue": {
                "signature": "DecodeValue(v reflect.Value) error",
                "description": "Decodes into reflect.Value. Unsafe deserialization sink.",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": [],
    },
    # =====================================================================
    # Web frameworks — third-party routers
    # =====================================================================
    "GoChiRouter": {
        "description": (
            "Chi HTTP router (chi.Router and chi.Mux). Path parameters extracted via URLParam "
            "are taint sources. Chi is one of the most popular lightweight routers in the Go ecosystem."
        ),
        "category": "web-frameworks",
        "go_mod": "require github.com/go-chi/chi/v5 v5.0.12",
        "methods": {
            "URLParam": {
                "signature": "URLParam(r *http.Request, key string) string",
                "description": "Returns URL path parameter for the given key (e.g. /users/{id}). User-controlled.",
                "role": "source",
                "tracks": ["return"],
            },
            "URLParamFromCtx": {
                "signature": "URLParamFromCtx(ctx context.Context, key string) string",
                "description": "Returns URL path parameter from the request context. User-controlled.",
                "role": "source",
                "tracks": ["return"],
            },
            "Handle": {
                "signature": "Handle(pattern string, h http.Handler)",
                "description": "Registers an http.Handler for a URL pattern. Routing primitive (neutral).",
                "role": "neutral",
                "tracks": [],
            },
            "HandleFunc": {
                "signature": "HandleFunc(pattern string, h http.HandlerFunc)",
                "description": "Registers a handler function for a pattern. Routing primitive (neutral).",
                "role": "neutral",
                "tracks": [],
            },
            "Get": {
                "signature": "Get(pattern string, h http.HandlerFunc)",
                "description": "Registers a GET handler for a pattern. Routing primitive (neutral).",
                "role": "neutral",
                "tracks": [],
            },
            "Post": {
                "signature": "Post(pattern string, h http.HandlerFunc)",
                "description": "Registers a POST handler. Routing primitive (neutral).",
                "role": "neutral",
                "tracks": [],
            },
        },
        "rules_using": ["GO-GORM-SQLI-001"],
    },
    "GoGorillaMuxRouter": {
        "description": (
            "Gorilla mux HTTP router (mux.Router). Path variables extracted via mux.Vars(r) are "
            "taint sources. Gorilla mux is the canonical router for larger Go web applications."
        ),
        "category": "web-frameworks",
        "go_mod": "require github.com/gorilla/mux v1.8.1",
        "methods": {
            "Vars": {
                "signature": "Vars(r *http.Request) map[string]string",
                "description": "Returns the route variables for the current request. All map values are user-controlled.",
                "role": "source",
                "tracks": ["return"],
            },
            "CurrentRoute": {
                "signature": "CurrentRoute(r *http.Request) *Route",
                "description": "Returns the matched route for the request. Metadata accessor (neutral).",
                "role": "neutral",
                "tracks": [],
            },
            "HandleFunc": {
                "signature": "HandleFunc(path string, f func(http.ResponseWriter, *http.Request)) *Route",
                "description": "Registers a handler function for a path. Routing primitive (neutral).",
                "role": "neutral",
                "tracks": [],
            },
            "Handle": {
                "signature": "Handle(path string, handler http.Handler) *Route",
                "description": "Registers an http.Handler for a path. Routing primitive (neutral).",
                "role": "neutral",
                "tracks": [],
            },
            "PathPrefix": {
                "signature": "PathPrefix(tpl string) *Route",
                "description": "Registers a sub-router under a path prefix. Routing primitive (neutral).",
                "role": "neutral",
                "tracks": [],
            },
        },
        "rules_using": [],
    },
    # =====================================================================
    # Databases — third-party drivers
    # =====================================================================
    "GoPgxConn": {
        "description": (
            "pgx PostgreSQL driver. Connection and Pool types expose Query/Exec/QueryRow that "
            "accept raw SQL strings — injection sinks when the SQL is built from user input. "
            "pgx is the recommended Postgres driver for new Go projects."
        ),
        "category": "databases",
        "go_mod": "require github.com/jackc/pgx/v5 v5.5.5",
        "methods": {
            "Exec": {
                "signature": "Exec(ctx context.Context, sql string, args ...any) (CommandTag, error)",
                "description": "Executes SQL that doesn't return rows. Sink when sql is built from user input.",
                "role": "sink",
                "tracks": [1],
            },
            "Query": {
                "signature": "Query(ctx context.Context, sql string, args ...any) (Rows, error)",
                "description": "Executes a query returning rows. SQL injection sink.",
                "role": "sink",
                "tracks": [1],
            },
            "QueryRow": {
                "signature": "QueryRow(ctx context.Context, sql string, args ...any) Row",
                "description": "Executes a query returning a single row. SQL injection sink.",
                "role": "sink",
                "tracks": [1],
            },
            "ExecEx": {
                "signature": "ExecEx(ctx context.Context, sql string, options *QueryExOptions, args ...any) (CommandTag, error)",
                "description": "pgx v4 compatibility shim for Exec. Same injection risk.",
                "role": "sink",
                "tracks": [1],
            },
            "QueryEx": {
                "signature": "QueryEx(ctx context.Context, sql string, options *QueryExOptions, args ...any) (*Rows, error)",
                "description": "pgx v4 compatibility shim for Query. Same injection risk.",
                "role": "sink",
                "tracks": [1],
            },
            "QueryRowEx": {
                "signature": "QueryRowEx(ctx context.Context, sql string, options *QueryExOptions, args ...any) *Row",
                "description": "pgx v4 compatibility shim for QueryRow. Same injection risk.",
                "role": "sink",
                "tracks": [1],
            },
            "SendBatch": {
                "signature": "SendBatch(ctx context.Context, b *Batch) BatchResults",
                "description": "Sends a batch of queries. Each query in the batch can be an injection sink.",
                "role": "sink",
                "tracks": [1],
            },
            "Prepare": {
                "signature": "Prepare(ctx context.Context, name, sql string) (*StatementDescription, error)",
                "description": "Creates a prepared statement. Sink when sql is user-controlled.",
                "role": "sink",
                "tracks": [2],
            },
        },
        "rules_using": ["GO-SQLI-002"],
    },
    "GoMongoCollection": {
        "description": (
            "MongoDB Go driver Collection and Client. Queries built from user input via "
            "bson.D or bson.M with string interpolation are NoSQL injection sinks. The filter "
            "argument on Find/Update/Delete operations is where tainted input lands."
        ),
        "category": "databases",
        "go_mod": "require go.mongodb.org/mongo-driver v1.14.0",
        "methods": {
            "Find": {
                "signature": "Find(ctx context.Context, filter any, opts ...*options.FindOptions) (*Cursor, error)",
                "description": "Queries documents matching filter. NoSQL injection sink if filter is built from user input.",
                "role": "sink",
                "tracks": [1],
            },
            "FindOne": {
                "signature": "FindOne(ctx context.Context, filter any, opts ...*options.FindOneOptions) *SingleResult",
                "description": "Returns one document matching filter. Same NoSQL injection risk.",
                "role": "sink",
                "tracks": [1],
            },
            "UpdateOne": {
                "signature": "UpdateOne(ctx context.Context, filter, update any, opts ...*options.UpdateOptions) (*UpdateResult, error)",
                "description": "Updates one document matching filter. Both filter and update are injection sinks.",
                "role": "sink",
                "tracks": [1],
            },
            "UpdateMany": {
                "signature": "UpdateMany(ctx context.Context, filter, update any, opts ...*options.UpdateOptions) (*UpdateResult, error)",
                "description": "Updates all matching documents. Injection sink on filter and update arguments.",
                "role": "sink",
                "tracks": [1],
            },
            "DeleteOne": {
                "signature": "DeleteOne(ctx context.Context, filter any, opts ...*options.DeleteOptions) (*DeleteResult, error)",
                "description": "Deletes first document matching filter. NoSQL injection sink.",
                "role": "sink",
                "tracks": [1],
            },
            "DeleteMany": {
                "signature": "DeleteMany(ctx context.Context, filter any, opts ...*options.DeleteOptions) (*DeleteResult, error)",
                "description": "Deletes all matching documents. NoSQL injection sink.",
                "role": "sink",
                "tracks": [1],
            },
            "Aggregate": {
                "signature": "Aggregate(ctx context.Context, pipeline any, opts ...*options.AggregateOptions) (*Cursor, error)",
                "description": "Runs an aggregation pipeline. Each stage can be an injection sink if built from user input.",
                "role": "sink",
                "tracks": [1],
            },
            "InsertOne": {
                "signature": "InsertOne(ctx context.Context, document any, opts ...*options.InsertOneOptions) (*InsertOneResult, error)",
                "description": "Inserts a document. Generally safe because fields are typed, but tainted document fields reach storage.",
                "role": "neutral",
                "tracks": [],
            },
        },
        "rules_using": [],
    },
    "GoRedisClient": {
        "description": (
            "go-redis Client for Redis operations. Most Redis commands are typed and safe, but "
            "Eval() and EvalSha() accept Lua scripts that can be injection sinks when the script "
            "body is user-controlled. ACL commands can also be sinks."
        ),
        "category": "databases",
        "go_mod": "require github.com/redis/go-redis/v9 v9.5.1",
        "methods": {
            "Eval": {
                "signature": "Eval(ctx context.Context, script string, keys []string, args ...any) *Cmd",
                "description": "Executes a Lua script on the Redis server. Injection sink if script is user-controlled.",
                "role": "sink",
                "tracks": [1],
            },
            "EvalSha": {
                "signature": "EvalSha(ctx context.Context, sha1 string, keys []string, args ...any) *Cmd",
                "description": "Executes a cached Lua script by SHA. Less risky than Eval but tainted SHA can still trigger unintended scripts.",
                "role": "sink",
                "tracks": [1],
            },
            "ScriptLoad": {
                "signature": "ScriptLoad(ctx context.Context, script string) *StringCmd",
                "description": "Registers a Lua script for later EvalSha. Sink when script is user-controlled.",
                "role": "sink",
                "tracks": [1],
            },
            "Do": {
                "signature": "Do(ctx context.Context, args ...any) *Cmd",
                "description": "Sends an arbitrary command. Command-injection sink when the command name is user-controlled.",
                "role": "sink",
                "tracks": [1],
            },
            "Get": {
                "signature": "Get(ctx context.Context, key string) *StringCmd",
                "description": "Fetches a string value. Source when cached data originated from user input.",
                "role": "source",
                "tracks": ["return"],
            },
            "Set": {
                "signature": "Set(ctx context.Context, key string, value any, expiration time.Duration) *StatusCmd",
                "description": "Stores a value. Typed and generally safe.",
                "role": "neutral",
                "tracks": [],
            },
        },
        "rules_using": [],
    },
    # =====================================================================
    # Serialization & config — third-party
    # =====================================================================
    "GoYAMLDecoder": {
        "description": (
            "gopkg.in/yaml.v3 Decoder for YAML deserialization. Decode() hydrates arbitrary Go "
            "types from YAML input — a deserialization sink when the YAML source is user-controlled. "
            "Package-level yaml.Unmarshal has the same properties."
        ),
        "category": "auth-config",
        "go_mod": "require gopkg.in/yaml.v3 v3.0.1",
        "methods": {
            "Decode": {
                "signature": "Decode(v any) error",
                "description": "Deserializes the next YAML document into v. Sink when the underlying reader is user-controlled.",
                "role": "sink",
                "tracks": [0],
            },
            "KnownFields": {
                "signature": "KnownFields(enable bool)",
                "description": "Configures the decoder to error on unknown fields. Hardening control (neutral).",
                "role": "neutral",
                "tracks": [],
            },
        },
        "rules_using": [],
    },
    "GoViperConfig": {
        "description": (
            "github.com/spf13/viper is the de-facto Go configuration library. Values returned from "
            "Get* methods are sources when the config file itself contains untrusted fields "
            "(environment, remote KV stores). Write methods that persist config back are typically neutral."
        ),
        "category": "auth-config",
        "go_mod": "require github.com/spf13/viper v1.18.2",
        "methods": {
            "Get": {
                "signature": "Get(key string) any",
                "description": "Returns the raw value for key. Source when the backing config contains user input.",
                "role": "source",
                "tracks": ["return"],
            },
            "GetString": {
                "signature": "GetString(key string) string",
                "description": "Returns the config value coerced to string. Source for user-supplied config.",
                "role": "source",
                "tracks": ["return"],
            },
            "GetInt": {
                "signature": "GetInt(key string) int",
                "description": "Returns the config value coerced to int. Numeric coercion acts as a sanitizer for SQL / path injection.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
            "GetBool": {
                "signature": "GetBool(key string) bool",
                "description": "Returns the config value coerced to bool. Sanitizer via type coercion.",
                "role": "sanitizer",
                "tracks": ["return"],
            },
            "GetStringSlice": {
                "signature": "GetStringSlice(key string) []string",
                "description": "Returns the config value as a string slice. Elements are sources.",
                "role": "source",
                "tracks": ["return"],
            },
            "ReadConfig": {
                "signature": "ReadConfig(in io.Reader) error",
                "description": "Reads config from a reader. Subsequent Get* values become sources if the reader is user-controlled.",
                "role": "neutral",
                "tracks": [],
            },
            "Unmarshal": {
                "signature": "Unmarshal(rawVal any, opts ...DecoderConfigOption) error",
                "description": "Hydrates a Go struct from the config. rawVal becomes tainted if the config contains user input.",
                "role": "source",
                "tracks": [0],
            },
        },
        "rules_using": [],
    },
    # =====================================================================
    # gRPC
    # =====================================================================
    "GoGRPCServerTransportStream": {
        "description": (
            "google.golang.org/grpc.ServerTransportStream exposes transport-layer metadata for "
            "in-flight gRPC calls. Method() returns the fully-qualified gRPC method name — path-like "
            "and frequently user-influenced via client-supplied routing. Header/Trailer methods ship "
            "metadata back to the client."
        ),
        "category": "auth-config",
        "go_mod": "require google.golang.org/grpc v1.62.1",
        "methods": {
            "Method": {
                "signature": "Method() string",
                "description": "Returns the fully-qualified gRPC method name. Source when the method path is used for authorization decisions.",
                "role": "source",
                "tracks": ["return"],
            },
            "SetHeader": {
                "signature": "SetHeader(md metadata.MD) error",
                "description": "Sets response headers. Neutral for outbound metadata, but secret leakage possible if md contains sensitive fields.",
                "role": "neutral",
                "tracks": [],
            },
            "SendHeader": {
                "signature": "SendHeader(md metadata.MD) error",
                "description": "Sends response headers immediately. Same considerations as SetHeader.",
                "role": "neutral",
                "tracks": [],
            },
            "SetTrailer": {
                "signature": "SetTrailer(md metadata.MD)",
                "description": "Sets response trailers. Neutral.",
                "role": "neutral",
                "tracks": [],
            },
        },
        "rules_using": [],
    },
    # =====================================================================
    # stdlib (late additions)
    # =====================================================================
    "GoIO": {
        "description": (
            "The io standard library package. ReadAll and Copy move data from readers — sources when "
            "the underlying reader is user-controlled (e.g. an http.Request.Body). WriteString writes "
            "to a writer and is a sink when the writer is an HTTP response."
        ),
        "category": "stdlib",
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "ReadAll": {
                "signature": "ReadAll(r Reader) ([]byte, error)",
                "description": "Reads from r until EOF and returns the result. Source when r wraps user input.",
                "role": "source",
                "tracks": ["return"],
            },
            "Copy": {
                "signature": "Copy(dst Writer, src Reader) (written int64, err error)",
                "description": "Copies from src to dst. Neutral data-transfer primitive; taint transits src → dst.",
                "role": "neutral",
                "tracks": [],
            },
            "CopyN": {
                "signature": "CopyN(dst Writer, src Reader, n int64) (written int64, err error)",
                "description": "Copies exactly n bytes from src to dst. Same as Copy.",
                "role": "neutral",
                "tracks": [],
            },
            "WriteString": {
                "signature": "WriteString(w Writer, s string) (n int, err error)",
                "description": "Writes s to w. Sink when w is a response writer and s is user-controlled (XSS).",
                "role": "sink",
                "tracks": [1],
            },
            "ReadFull": {
                "signature": "ReadFull(r Reader, buf []byte) (n int, err error)",
                "description": "Reads exactly len(buf) bytes from r. Buffer becomes tainted if r is user-controlled.",
                "role": "source",
                "tracks": [1],
            },
            "NopCloser": {
                "signature": "NopCloser(r Reader) ReadCloser",
                "description": "Wraps r in a no-op ReadCloser. Neutral transformation.",
                "role": "neutral",
                "tracks": [],
            },
        },
        "rules_using": [],
    },
    "GoJSON": {
        "description": (
            "encoding/json for JSON encode/decode. Unmarshal and Decoder.Decode deserialize JSON into "
            "Go values — the destination struct becomes tainted if the input bytes are user-controlled. "
            "Encoder.Encode writes JSON to a writer, a sink when the writer is an HTTP response."
        ),
        "category": "stdlib",
        "go_mod": "// standard library — no go.mod entry required",
        "methods": {
            "Marshal": {
                "signature": "Marshal(v any) ([]byte, error)",
                "description": "Serializes v to JSON bytes. Generally neutral.",
                "role": "neutral",
                "tracks": [],
            },
            "Unmarshal": {
                "signature": "Unmarshal(data []byte, v any) error",
                "description": "Parses JSON bytes into v. v becomes tainted if data comes from user input.",
                "role": "source",
                "tracks": [1],
            },
            "NewDecoder": {
                "signature": "NewDecoder(r io.Reader) *Decoder",
                "description": "Creates a streaming decoder bound to r. Decoder.Decode is the actual source.",
                "role": "neutral",
                "tracks": [],
            },
            "NewEncoder": {
                "signature": "NewEncoder(w io.Writer) *Encoder",
                "description": "Creates a streaming encoder bound to w. Encoder.Encode is the actual sink.",
                "role": "neutral",
                "tracks": [],
            },
            "Decode": {
                "signature": "Decode(v any) error",
                "description": "Reads the next JSON-encoded value from the stream into v. Source when stream is user input.",
                "role": "source",
                "tracks": [0],
            },
            "Encode": {
                "signature": "Encode(v any) error",
                "description": "Writes v as JSON to the underlying writer. Sink when writer is a response and v contains raw HTML.",
                "role": "sink",
                "tracks": [0],
            },
        },
        "rules_using": [],
    },
}
