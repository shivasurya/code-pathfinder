"""Tests for JSON IR compilation."""

import json
import os
import tempfile
from rules.container_decorators import (
    dockerfile_rule,
    compose_rule,
    clear_rules,
)
from rules.container_matchers import instruction, missing, service_has
from rules.container_ir import (
    compile_dockerfile_rules,
    compile_compose_rules,
    compile_all_rules,
    compile_to_json,
    write_ir_file,
)


class TestIRCompilation:
    def setup_method(self):
        clear_rules()

    def test_compile_dockerfile_rules(self):
        @dockerfile_rule(id="TEST-001", severity="HIGH", cwe="CWE-250")
        def test_rule():
            return missing(instruction="USER")

        compiled = compile_dockerfile_rules()
        assert len(compiled) == 1
        assert compiled[0]["id"] == "TEST-001"
        assert compiled[0]["severity"] == "HIGH"
        assert compiled[0]["cwe"] == "CWE-250"
        assert compiled[0]["rule_type"] == "dockerfile"
        assert compiled[0]["matcher"]["type"] == "missing_instruction"

    def test_compile_compose_rules(self):
        @compose_rule(id="COMPOSE-001")
        def priv_rule():
            return service_has(key="privileged", equals=True)

        compiled = compile_compose_rules()
        assert len(compiled) == 1
        assert compiled[0]["rule_type"] == "compose"

    def test_compile_all_rules(self):
        @dockerfile_rule(id="D-001")
        def d_rule():
            return missing(instruction="USER")

        @compose_rule(id="C-001")
        def c_rule():
            return service_has(key="privileged", equals=True)

        compiled = compile_all_rules()
        assert "dockerfile" in compiled
        assert "compose" in compiled
        assert len(compiled["dockerfile"]) == 1
        assert len(compiled["compose"]) == 1

    def test_compile_to_json(self):
        @dockerfile_rule(id="JSON-001")
        def json_rule():
            return instruction(type="FROM", image_tag="latest")

        json_str = compile_to_json()
        parsed = json.loads(json_str)
        assert "dockerfile" in parsed

    def test_compile_to_json_compact(self):
        @dockerfile_rule(id="COMPACT-001")
        def compact_rule():
            return missing(instruction="USER")

        json_str = compile_to_json(pretty=False)
        parsed = json.loads(json_str)
        assert "dockerfile" in parsed
        # Compact should have no indentation
        assert "\n  " not in json_str

    def test_write_ir_file(self):
        @dockerfile_rule(id="FILE-001")
        def file_rule():
            return missing(instruction="USER")

        with tempfile.NamedTemporaryFile(mode="w", delete=False, suffix=".json") as f:
            filepath = f.name

        try:
            write_ir_file(filepath, pretty=True)

            with open(filepath, "r") as f:
                content = f.read()
                parsed = json.loads(content)
                assert "dockerfile" in parsed
                assert len(parsed["dockerfile"]) == 1
        finally:
            os.unlink(filepath)
