"""Tests for programmatic access."""

from rules.container_programmatic import (
    custom_check,
    ProgrammaticMatcher,
    DockerfileAccess,
    ComposeAccess,
)


class TestCustomCheck:
    def test_basic(self):
        def my_check(dockerfile):
            return dockerfile.is_running_as_root()

        m = custom_check(my_check, "Check for root user")
        d = m.to_dict()
        assert d["type"] == "programmatic"
        assert d["has_callable"] is True
        assert d["description"] == "Check for root user"

    def test_callable_stored(self):
        def my_check(dockerfile):
            return True

        m = custom_check(my_check)
        assert callable(m.check_function)

    def test_without_description(self):
        def my_check(compose):
            return len(compose.get_privileged_services()) > 0

        m = custom_check(my_check)
        d = m.to_dict()
        assert d["description"] == ""

    def test_complex_function(self):
        def check_multi_stage(dockerfile):
            if not dockerfile.is_multi_stage():
                return False
            final = dockerfile.get_final_stage()
            return final.user_name == "root"

        m = custom_check(check_multi_stage, "Multi-stage runs as root")
        assert m.check_function is check_multi_stage
        assert m.description == "Multi-stage runs as root"


class TestProgrammaticMatcher:
    def test_creation(self):
        def func():
            return True

        pm = ProgrammaticMatcher(check_function=func, description="Test")
        assert pm.check_function is func
        assert pm.description == "Test"

    def test_to_dict(self):
        pm = ProgrammaticMatcher(
            check_function=lambda x: True, description="Lambda test"
        )
        d = pm.to_dict()
        assert d["type"] == "programmatic"
        assert d["has_callable"] is True
        assert d["description"] == "Lambda test"


class MockDockerfileGraph:
    """Mock for testing DockerfileAccess."""

    def __init__(self):
        self.instructions = {"USER": [{"user_name": "root"}]}

    def GetInstructions(self, instruction_type):
        return self.instructions.get(instruction_type, [])

    def HasInstruction(self, instruction_type):
        return instruction_type in self.instructions

    def GetFinalUser(self):
        return {"user_name": "root"}

    def IsRunningAsRoot(self):
        return True

    def GetStages(self):
        return [{"alias": "builder"}]

    def IsMultiStage(self):
        return True

    def GetStageByAlias(self, alias):
        return {"alias": alias}

    def GetFinalStage(self):
        return {"alias": "final"}


class TestDockerfileAccess:
    def test_get_instructions(self):
        graph = MockDockerfileGraph()
        access = DockerfileAccess(graph)
        result = access.get_instructions("USER")
        assert len(result) == 1

    def test_has_instruction(self):
        graph = MockDockerfileGraph()
        access = DockerfileAccess(graph)
        assert access.has_instruction("USER") is True
        assert access.has_instruction("HEALTHCHECK") is False

    def test_get_final_user(self):
        graph = MockDockerfileGraph()
        access = DockerfileAccess(graph)
        user = access.get_final_user()
        assert user["user_name"] == "root"

    def test_is_running_as_root(self):
        graph = MockDockerfileGraph()
        access = DockerfileAccess(graph)
        assert access.is_running_as_root() is True

    def test_get_stages(self):
        graph = MockDockerfileGraph()
        access = DockerfileAccess(graph)
        stages = access.get_stages()
        assert len(stages) == 1

    def test_is_multi_stage(self):
        graph = MockDockerfileGraph()
        access = DockerfileAccess(graph)
        assert access.is_multi_stage() is True

    def test_get_stage_by_alias(self):
        graph = MockDockerfileGraph()
        access = DockerfileAccess(graph)
        stage = access.get_stage_by_alias("builder")
        assert stage["alias"] == "builder"

    def test_get_final_stage(self):
        graph = MockDockerfileGraph()
        access = DockerfileAccess(graph)
        stage = access.get_final_stage()
        assert stage["alias"] == "final"


class MockComposeGraph:
    """Mock for testing ComposeAccess."""

    def __init__(self):
        self.services = ["web", "db"]

    def GetServices(self):
        return self.services

    def ServiceHas(self, service_name, key, value):
        return service_name == "web" and key == "privileged" and value is True

    def ServiceGet(self, service_name, key):
        if service_name == "web" and key == "image":
            return "nginx"
        return None

    def GetPrivilegedServices(self):
        return ["web"]

    def ServicesWithDockerSocket(self):
        return ["db"]

    def ServicesWithHostNetwork(self):
        return []


class TestComposeAccess:
    def test_get_services(self):
        graph = MockComposeGraph()
        access = ComposeAccess(graph)
        services = access.get_services()
        assert len(services) == 2
        assert "web" in services

    def test_service_has(self):
        graph = MockComposeGraph()
        access = ComposeAccess(graph)
        assert access.service_has("web", "privileged", True) is True
        assert access.service_has("db", "privileged", True) is False

    def test_service_get(self):
        graph = MockComposeGraph()
        access = ComposeAccess(graph)
        image = access.service_get("web", "image")
        assert image == "nginx"

    def test_get_privileged_services(self):
        graph = MockComposeGraph()
        access = ComposeAccess(graph)
        privileged = access.get_privileged_services()
        assert privileged == ["web"]

    def test_services_with_docker_socket(self):
        graph = MockComposeGraph()
        access = ComposeAccess(graph)
        with_socket = access.services_with_docker_socket()
        assert with_socket == ["db"]

    def test_services_with_host_network(self):
        graph = MockComposeGraph()
        access = ComposeAccess(graph)
        host_network = access.services_with_host_network()
        assert len(host_network) == 0
