from rules.python_decorators import python_rule
from codepathfinder import calls, flows, QueryType
from codepathfinder.presets import PropagationPresets


class OSModule(QueryType):
    fqns = ["os"]


class OSPathModule(QueryType):
    fqns = ["os.path"]


class ShutilModule(QueryType):
    fqns = ["shutil"]


class Builtins(QueryType):
    fqns = ["builtins"]


@python_rule(
    id="PYTHON-LANG-SEC-162",
    name="Symlink Following Arbitrary File Access",
    severity="HIGH",
    category="lang",
    cwe="CWE-59",
    tags="python,symlink,path-traversal,file-access,OWASP-A01,CWE-59",
    message="User-controlled path flows to file operation without symlink resolution. "
            "Attacker can craft symbolic links to read/write arbitrary files. "
            "Resolve symlinks with os.path.realpath() and validate with is_relative_to() before access.",
    owasp="A01:2021",
)
def detect_symlink_following():
    """Detects user-controlled paths flowing to file operations without symlink resolution.
    CVE: GHSA-g925-f788-4jh7 (Weblate arbitrary file read via symlinks).

    Source: os.path.join (type-inferred), request.args/form.get
    Sink: open(), os.readlink(), shutil.copy/move (all type-inferred via stdlib)
    Sanitizer: os.path.realpath() (type-inferred)
    """
    return flows(
        from_sources=[
            OSPathModule.method("join"),
            calls("request.args.get"),
            calls("request.form.get"),
            calls("*.get"),
        ],
        to_sinks=[
            Builtins.method("open").tracks(0),
            OSModule.method("readlink").tracks(0),
            OSModule.method("symlink"),
            ShutilModule.method("copy", "copy2", "move", "copyfile").tracks(0),
        ],
        sanitized_by=[
            OSPathModule.method("realpath"),
            OSPathModule.method("abspath"),
            calls("*.resolve"),
            calls("*.is_relative_to"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="global",
    )
