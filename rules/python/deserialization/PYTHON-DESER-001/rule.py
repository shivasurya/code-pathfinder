from rules.python_decorators import python_rule
from codepathfinder import calls, flows
from codepathfinder.presets import PropagationPresets


@python_rule(
    id="PYTHON-DESER-001",
    name="Unsafe Pickle Deserialization",
    severity="CRITICAL",
    category="deserialization",
    cwe="CWE-502",
    cve="CVE-2021-3177",
    tags="python,deserialization,pickle,rce,untrusted-data,OWASP-A08,CWE-502,remote-code-execution,critical,security,intra-procedural",
    message="Unsafe pickle deserialization: Untrusted data flows to pickle.loads() which can execute arbitrary code. Use json.loads() instead.",
    owasp="A08:2021",
)
def detect_pickle_deserialization():
    """
    Detects unsafe pickle deserialization where user input flows to pickle.loads() within a single function.

    LIMITATION: Only detects intra-procedural flows (within one function).
    Will NOT detect if user input is in one function and pickle.loads is in another.

    Example vulnerable code:
        user_data = request.data
        obj = pickle.loads(user_data)  # RCE!
    """
    return flows(
        from_sources=[
            calls("request.data"),
            calls("request.get_data"),
            calls("request.GET"),
            calls("request.POST"),
            calls("request.COOKIES"),
            calls("input"),
            calls("*.data"),
            calls("*.GET"),
            calls("*.POST"),
            calls("*.read"),
            calls("*.recv"),
        ],
        to_sinks=[
            calls("pickle.loads"),
            calls("pickle.load"),
            calls("_pickle.loads"),
            calls("_pickle.load"),
            calls("*.loads"),
            calls("*.load"),
        ],
        sanitized_by=[
            calls("*.validate"),
            calls("*.verify_signature"),
            calls("*.verify"),
            calls("hmac.compare_digest"),
        ],
        propagates_through=PropagationPresets.standard(),
        scope="local",  # CRITICAL: Only intra-procedural analysis works
    )
