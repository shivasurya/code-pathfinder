from codepathfinder import calls, flows, rule
from codepathfinder.presets import PropagationPresets

@rule(id="TEST-LOW-001", severity="low")
def low_test_rule():
    """Low test detection for integration tests"""
    return flows(
        from_sources=[calls("request.get_data"), calls("*.get_data")],
        to_sinks=[calls("pickle.loads"), calls("*.loads")],
        propagates_through=PropagationPresets.standard(),
        scope="local"
    )
