"""Container security rules DSL for Dockerfile and docker-compose."""

from .container_decorators import dockerfile_rule, compose_rule
from .container_matchers import instruction, missing, service_has, service_missing
from .container_ir import compile_all_rules, compile_to_json
from .container_combinators import (
    all_of,
    any_of,
    none_of,
    instruction_after,
    instruction_before,
    stage,
    final_stage_has,
)
from .container_programmatic import custom_check, DockerfileAccess, ComposeAccess

__all__ = [
    "dockerfile_rule",
    "compose_rule",
    "instruction",
    "missing",
    "service_has",
    "service_missing",
    "compile_all_rules",
    "compile_to_json",
    "all_of",
    "any_of",
    "none_of",
    "instruction_after",
    "instruction_before",
    "stage",
    "final_stage_has",
    "custom_check",
    "DockerfileAccess",
    "ComposeAccess",
]
