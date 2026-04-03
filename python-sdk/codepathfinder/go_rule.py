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
