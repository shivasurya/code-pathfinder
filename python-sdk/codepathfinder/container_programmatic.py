"""
Programmatic access to Dockerfile and docker-compose objects.
"""

from typing import Callable, Dict, Any
from dataclasses import dataclass


@dataclass
class ProgrammaticMatcher:
    """Wraps a custom validation function."""

    check_function: Callable
    description: str = ""

    def to_dict(self) -> Dict[str, Any]:
        return {
            "type": "programmatic",
            "has_callable": True,
            "description": self.description,
        }


def custom_check(check: Callable, description: str = "") -> ProgrammaticMatcher:
    """
    Create a custom validation function.

    The check function receives the parsed dockerfile or compose object
    and should return True if the rule matches (vulnerability found).

    Example:
        @dockerfile_rule(id="DOCKER-CUSTOM-001")
        def last_user_is_root():
            def check(dockerfile):
                final_user = dockerfile.get_final_user()
                return final_user is None or final_user.user_name == "root"
            return custom_check(check, "Check if last USER is root")
    """
    return ProgrammaticMatcher(check_function=check, description=description)


class DockerfileAccess:
    """
    Provides programmatic access to Dockerfile structure.
    Used in custom validation functions.
    """

    def __init__(self, dockerfile_graph):
        self._graph = dockerfile_graph

    def get_instructions(self, instruction_type: str):
        """Get all instructions of a type."""
        return self._graph.GetInstructions(instruction_type)

    def has_instruction(self, instruction_type: str) -> bool:
        """Check if instruction type exists."""
        return self._graph.HasInstruction(instruction_type)

    def get_final_user(self):
        """Get the last USER instruction."""
        return self._graph.GetFinalUser()

    def is_running_as_root(self) -> bool:
        """Check if container runs as root."""
        return self._graph.IsRunningAsRoot()

    def get_stages(self):
        """Get all build stages."""
        return self._graph.GetStages()

    def is_multi_stage(self) -> bool:
        """Check if Dockerfile uses multi-stage build."""
        return self._graph.IsMultiStage()

    def get_stage_by_alias(self, alias: str):
        """Get a stage by its AS alias."""
        return self._graph.GetStageByAlias(alias)

    def get_final_stage(self):
        """Get the final build stage."""
        return self._graph.GetFinalStage()


class ComposeAccess:
    """
    Provides programmatic access to docker-compose structure.
    Used in custom validation functions.
    """

    def __init__(self, compose_graph):
        self._graph = compose_graph

    def get_services(self):
        """Get all service names."""
        return self._graph.GetServices()

    def service_has(self, service_name: str, key: str, value) -> bool:
        """Check if service has property with value."""
        return self._graph.ServiceHas(service_name, key, value)

    def service_get(self, service_name: str, key: str):
        """Get service property value."""
        return self._graph.ServiceGet(service_name, key)

    def get_privileged_services(self):
        """Get services with privileged: true."""
        return self._graph.GetPrivilegedServices()

    def services_with_docker_socket(self):
        """Get services mounting Docker socket."""
        return self._graph.ServicesWithDockerSocket()

    def services_with_host_network(self):
        """Get services using host network mode."""
        return self._graph.ServicesWithHostNetwork()
