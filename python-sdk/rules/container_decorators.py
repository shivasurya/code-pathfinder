"""
Backward-compatibility shim. container_decorators has moved to the codepathfinder package.
Import from: from codepathfinder.container_decorators import dockerfile_rule, compose_rule
"""
from codepathfinder.container_decorators import (  # noqa: F401
    RuleMetadata,
    DockerfileRuleDefinition,
    ComposeRuleDefinition,
    dockerfile_rule,
    compose_rule,
    get_dockerfile_rules,
    get_compose_rules,
    clear_rules,
)
