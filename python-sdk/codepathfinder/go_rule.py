"""
Go QueryType classes for type-constrained security rule matching.

These classes define Go stdlib types that rule authors use to match
method calls and attribute access on typed receivers.

Usage:
    from codepathfinder.go_rule import GoHTTPRequest, GoSQLDB

    GoHTTPRequest.method("FormValue")       # matches r.FormValue(...)
    GoHTTPRequest.attr("URL.Path", "Host")  # matches r.URL.Path, r.Host
    GoSQLDB.method("Query", "Exec")         # matches db.Query(...), db.Exec(...)
"""

from .query_type import QueryType

# --- net/http ---


class GoHTTPRequest(QueryType):
    """*http.Request — HTTP handler request parameter."""

    fqns = ["net/http.Request"]
    patterns = ["*.Request"]
    match_subclasses = False


class GoHTTPClient(QueryType):
    """*http.Client and package-level http.Get/Post."""

    fqns = ["net/http.Client", "net/http"]
    patterns = ["http.Client"]
    match_subclasses = False


class GoHTTPResponseWriter(QueryType):
    """http.ResponseWriter — HTTP response sink."""

    fqns = ["net/http.ResponseWriter"]
    patterns = ["*.ResponseWriter"]
    match_subclasses = False


# --- database/sql ---


class GoSQLDB(QueryType):
    """*sql.DB, *sql.Tx, *sql.Stmt — database handles."""

    fqns = ["database/sql.DB", "database/sql.Tx", "database/sql.Stmt"]
    patterns = ["*.DB", "*.Tx"]
    match_subclasses = False


# --- os ---


class GoOS(QueryType):
    """os package — file operations, env vars."""

    fqns = ["os", "os.File"]
    patterns = ["os.*"]
    match_subclasses = False


# --- os/exec ---


class GoOSExec(QueryType):
    """os/exec — command execution."""

    fqns = ["os/exec", "os/exec.Cmd"]
    patterns = ["exec.*"]
    match_subclasses = False


# --- fmt ---


class GoFmt(QueryType):
    """fmt — string formatting (taint propagation source)."""

    fqns = ["fmt"]
    patterns = ["fmt.*"]
    match_subclasses = False


# --- io ---


class GoIO(QueryType):
    """io package — ReadAll, Copy, etc."""

    fqns = ["io"]
    patterns = ["io.*"]
    match_subclasses = False


# --- path/filepath ---


class GoFilepath(QueryType):
    """path/filepath — path sanitization."""

    fqns = ["path/filepath"]
    patterns = ["filepath.*"]
    match_subclasses = False


# --- strconv ---


class GoStrconv(QueryType):
    """strconv — type conversion sanitizers."""

    fqns = ["strconv"]
    patterns = ["strconv.*"]
    match_subclasses = False


# --- encoding/json ---


class GoJSON(QueryType):
    """encoding/json — JSON encode/decode."""

    fqns = ["encoding/json", "encoding/json.Decoder"]
    patterns = ["json.*"]
    match_subclasses = False


# --- html/template ---


class GoTemplate(QueryType):
    """html/template — template execution (auto-escapes by default)."""

    fqns = ["html/template.Template", "text/template.Template"]
    patterns = ["*.Template"]
    match_subclasses = False


# --- context ---


class GoContext(QueryType):
    """context.Context — request-scoped values."""

    fqns = ["context.Context"]
    patterns = ["*.Context"]
    match_subclasses = False


# --- crypto ---


class GoCrypto(QueryType):
    """crypto packages — hashing and encryption sanitizers."""

    fqns = ["crypto/sha256", "crypto/sha512", "crypto/hmac", "crypto/aes"]
    patterns = ["sha256.*", "sha512.*", "hmac.*"]
    match_subclasses = False


# ---------------------------------------------------------------------------
# Third-party Go framework QueryTypes
# ---------------------------------------------------------------------------

# --- gorm.io/gorm ---


class GoGormDB(QueryType):
    """GORM database handle — ORM for Go."""

    fqns = ["gorm.io/gorm.DB"]
    patterns = ["*.DB"]
    match_subclasses = False


# --- github.com/gin-gonic/gin ---


class GoGinContext(QueryType):
    """Gin HTTP framework request context."""

    fqns = ["github.com/gin-gonic/gin.Context"]
    patterns = ["*.Context"]
    match_subclasses = False


# --- github.com/labstack/echo/v4 ---


class GoEchoContext(QueryType):
    """Echo HTTP framework request context."""

    fqns = ["github.com/labstack/echo/v4.Context"]
    patterns = ["*.Context"]
    match_subclasses = False


# --- github.com/gofiber/fiber/v2 ---


class GoFiberCtx(QueryType):
    """Fiber HTTP framework request context."""

    fqns = ["github.com/gofiber/fiber/v2.Ctx"]
    patterns = ["*.Ctx"]
    match_subclasses = False


# --- google.golang.org/grpc ---


class GoGRPCServerTransportStream(QueryType):
    """gRPC server transport stream for metadata access."""

    fqns = ["google.golang.org/grpc.ServerTransportStream"]
    patterns = ["*.ServerTransportStream"]
    match_subclasses = False


# --- github.com/jackc/pgx/v5 ---


class GoPgxConn(QueryType):
    """pgx PostgreSQL driver connection and pool."""

    fqns = ["github.com/jackc/pgx/v5.Conn", "github.com/jackc/pgx/v5/pgxpool.Pool"]
    patterns = ["*.Conn", "*.Pool"]
    match_subclasses = False


# --- github.com/jmoiron/sqlx ---


class GoSqlxDB(QueryType):
    """sqlx extended database handle (DB and Tx)."""

    fqns = ["github.com/jmoiron/sqlx.DB", "github.com/jmoiron/sqlx.Tx"]
    patterns = ["*.DB", "*.Tx"]
    match_subclasses = False


# --- github.com/redis/go-redis/v9 ---


class GoRedisClient(QueryType):
    """go-redis client for Redis operations."""

    fqns = ["github.com/redis/go-redis/v9.Client"]
    patterns = ["*.Client"]
    match_subclasses = False


# --- go.mongodb.org/mongo-driver/mongo ---


class GoMongoCollection(QueryType):
    """MongoDB Go driver client and collection."""

    fqns = [
        "go.mongodb.org/mongo-driver/mongo.Collection",
        "go.mongodb.org/mongo-driver/mongo.Client",
    ]
    patterns = ["*.Collection", "*.Client"]
    match_subclasses = False


# --- github.com/golang-jwt/jwt/v5 ---


class GoJWTToken(QueryType):
    """golang-jwt token for JWT parsing and validation."""

    fqns = ["github.com/golang-jwt/jwt/v5.Token"]
    patterns = ["*.Token"]
    match_subclasses = False


# --- github.com/gorilla/mux ---


class GoGorillaMuxRouter(QueryType):
    """Gorilla mux HTTP router."""

    fqns = ["github.com/gorilla/mux.Router"]
    patterns = ["*.Router"]
    match_subclasses = False


# --- github.com/go-resty/resty/v2 ---


class GoRestyClient(QueryType):
    """Resty HTTP client (Client and Request)."""

    fqns = ["github.com/go-resty/resty/v2.Client", "github.com/go-resty/resty/v2.Request"]
    patterns = ["*.Client", "*.Request"]
    match_subclasses = False


# --- github.com/go-chi/chi/v5 ---


class GoChiRouter(QueryType):
    """Chi HTTP router (Router and Mux)."""

    fqns = ["github.com/go-chi/chi/v5.Router", "github.com/go-chi/chi/v5.Mux"]
    patterns = ["*.Router", "*.Mux"]
    match_subclasses = False


# --- github.com/spf13/viper ---


class GoViperConfig(QueryType):
    """Viper configuration management."""

    fqns = ["github.com/spf13/viper.Viper"]
    patterns = ["*.Viper"]
    match_subclasses = False


# --- gopkg.in/yaml.v3 ---


class GoYAMLDecoder(QueryType):
    """YAML v3 decoder for deserialization."""

    fqns = ["gopkg.in/yaml.v3.Decoder"]
    patterns = ["*.Decoder"]
    match_subclasses = False
