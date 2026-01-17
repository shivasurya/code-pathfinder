from codepathfinder import calls, flows, rule
from codepathfinder.presets import PropagationPresets

@rule(id="TEST-HIGH-001", severity="high")
def high_test_rule():
    """High test detection for integration tests"""
    return flows(
        from_sources=[calls("request.get_data"), calls("*.get_data")],
        to_sinks=[calls("pickle.loads"), calls("*.loads")],
        propagates_through=PropagationPresets.standard(),
        scope="local"
    )
