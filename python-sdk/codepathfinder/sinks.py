"""
Semantic sink categories for taint analysis.

Pre-built sink matchers covering common frameworks (Django, Flask, stdlib).
Each function returns a composable matcher that can be used in flows().

Example:
    from codepathfinder import sinks, flows, sources

    flows(
        from_sources=sources.http_input(),
        to_sinks=sinks.sql_execution(),
    )
"""

from .matchers import calls, calls_on
from .logic import Or


def sql_execution():
    """SQL execution sinks — type-aware.

    Covers:
    - sqlite3: Cursor.execute, Cursor.executemany, Connection.execute
    - SQLAlchemy: Engine.execute, Session.execute
    - Django ORM: QuerySet.raw, QuerySet.extra, RawSQL()
    """
    return Or(
        # sqlite3 / DB-API
        calls_on("Cursor", "execute", fallback="none"),
        calls_on("Cursor", "executemany", fallback="none"),
        calls_on("Connection", "execute", fallback="none"),
        # SQLAlchemy
        calls_on("Engine", "execute", fallback="none"),
        calls_on("Session", "execute", fallback="none"),
        # Django ORM raw
        calls_on("QuerySet", "raw", fallback="none"),
        calls_on("QuerySet", "extra", fallback="none"),
        calls("RawSQL"),
    )


def command_execution():
    """OS command execution sinks.

    Covers:
    - os: system, popen
    - subprocess: run, call, Popen, check_output, check_call
    """
    return Or(
        calls("os.system"),
        calls("os.popen"),
        calls("subprocess.run"),
        calls("subprocess.call"),
        calls("subprocess.Popen"),
        calls("subprocess.check_output"),
        calls("subprocess.check_call"),
    )


def code_execution():
    """Dynamic code execution sinks.

    Covers:
    - Built-in: eval(), exec(), compile(), __import__()
    """
    return Or(
        calls("eval"),
        calls("exec"),
        calls("compile"),
        calls("__import__"),
    )


def template_render():
    """Template rendering sinks (XSS risk).

    Covers:
    - Jinja2: Template.render, Environment.from_string
    - Django: Template.render, mark_safe
    - Generic: render_template_string
    """
    return Or(
        calls_on("Template", "render", fallback="none"),
        calls_on("Environment", "from_string", fallback="none"),
        calls("mark_safe"),
        calls("render_template_string"),
    )


def xpath_query():
    """XPath/XML query sinks (XXE/injection risk).

    Covers:
    - lxml: etree.parse, etree.fromstring, XPath evaluation
    - xml.etree: ElementTree.parse, fromstring
    """
    return Or(
        calls("lxml.etree.parse"),
        calls("lxml.etree.fromstring"),
        calls_on("XPath", "evaluate", fallback="none"),
        calls("xml.etree.ElementTree.parse"),
        calls("xml.etree.ElementTree.fromstring"),
    )


def ldap_query():
    """LDAP query sinks (injection risk).

    Covers:
    - ldap: search_s, search_st, search_ext_s
    """
    return Or(
        calls_on("LDAPObject", "search_s", fallback="none"),
        calls_on("LDAPObject", "search_st", fallback="none"),
        calls_on("LDAPObject", "search_ext_s", fallback="none"),
    )


def file_write():
    """File write sinks.

    Covers:
    - Built-in: write(), writelines()
    - pathlib: Path.write_text(), Path.write_bytes()
    """
    return Or(
        calls("write"),
        calls("writelines"),
        calls_on("Path", "write_text", fallback="none"),
        calls_on("Path", "write_bytes", fallback="none"),
    )


def file_open():
    """File open sinks (path traversal risk).

    Covers:
    - Built-in: open()
    - io: io.open()
    """
    return Or(
        calls("open"),
        calls("io.open"),
    )


def path_operation():
    """File path operations (path traversal risk).

    Covers:
    - os: os.remove, os.unlink, os.rename, os.chmod, os.mkdir
    - shutil: shutil.copy, shutil.move, shutil.rmtree
    - open() — file access with user-controlled paths
    """
    return Or(
        calls("os.remove"),
        calls("os.unlink"),
        calls("os.rename"),
        calls("os.chmod"),
        calls("os.mkdir"),
        calls("shutil.copy"),
        calls("shutil.move"),
        calls("shutil.rmtree"),
        calls("open"),
    )


def http_request():
    """HTTP request sinks (SSRF risk).

    Covers:
    - requests: get, post, put, delete, request
    - urllib: urlopen, Request
    - httpx: get, post
    """
    return Or(
        calls("requests.get"),
        calls("requests.post"),
        calls("requests.put"),
        calls("requests.delete"),
        calls("requests.request"),
        calls("urllib.request.urlopen"),
        calls("urllib.request.Request"),
        calls("httpx.get"),
        calls("httpx.post"),
    )


def socket_connect():
    """Socket connection sinks.

    Covers:
    - socket: socket.connect, socket.bind
    """
    return Or(
        calls_on("socket", "connect", fallback="none"),
        calls_on("socket", "bind", fallback="none"),
    )


def deserialize():
    """Deserialization sinks (insecure deserialization risk).

    Covers:
    - pickle: loads, load
    - yaml: load, unsafe_load
    - marshal: loads, load
    - jsonpickle: decode
    """
    return Or(
        calls("pickle.loads"),
        calls("pickle.load"),
        calls("yaml.load"),
        calls("yaml.unsafe_load"),
        calls("marshal.loads"),
        calls("marshal.load"),
        calls("jsonpickle.decode"),
    )
