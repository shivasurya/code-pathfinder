from codepathfinder import calls, flows, rule
from codepathfinder.presets import PropagationPresets

@rule(id="TEST-CRIT-001", severity="critical", cwe="CWE-502")
def critical_test_rule():
    """Critical test detection for integration tests"""
    return flows(
        from_sources=[calls("request.get_data"), calls("*.get_data")],
        to_sinks=[calls("pickle.loads"), calls("*.loads")],
        propagates_through=PropagationPresets.standard(),
        scope="local"
    )
