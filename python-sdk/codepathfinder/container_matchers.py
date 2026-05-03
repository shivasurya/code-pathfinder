"""
Matcher functions for Dockerfile and docker-compose rules.
"""

from typing import Optional, Any, List, Dict
from dataclasses import dataclass, field


@dataclass
class Matcher:
    """Base class for all matchers."""

    type: str
    params: Dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> Dict[str, Any]:
        """Convert matcher to dictionary for JSON IR."""
        return {"type": self.type, **self.params}


# --- Dockerfile Matchers ---


def instruction(
    type: str,
    # FROM instruction
    base_image: Optional[str] = None,
    image_tag: Optional[str] = None,
    image_tag_regex: Optional[str] = None,
    missing_digest: Optional[bool] = None,
    # USER instruction
    user_name: Optional[str] = None,
    user_name_regex: Optional[str] = None,
    # EXPOSE instruction
    port: Optional[int] = None,
    port_less_than: Optional[int] = None,
    port_greater_than: Optional[int] = None,
    protocol: Optional[str] = None,
    # ARG instruction
    arg_name: Optional[str] = None,
    arg_name_regex: Optional[str] = None,
    # COPY/ADD instruction
    copy_from: Optional[str] = None,
    chown: Optional[str] = None,
    missing_flag: Optional[str] = None,
    # HEALTHCHECK instruction
    healthcheck_interval_less_than: Optional[str] = None,
    healthcheck_timeout_greater_than: Optional[str] = None,
    healthcheck_retries_greater_than: Optional[int] = None,
    # LABEL instruction
    label_key: Optional[str] = None,
    label_value_regex: Optional[str] = None,
    # CMD/ENTRYPOINT
    command_form: Optional[str] = None,
    # WORKDIR
    workdir_not_absolute: Optional[bool] = None,
    # STOPSIGNAL
    signal_not_in: Optional[List[str]] = None,
    # Generic matchers
    contains: Optional[str] = None,
    not_contains: Optional[str] = None,
    regex: Optional[str] = None,
    not_regex: Optional[str] = None,
    # Custom validation
    validate: Optional[callable] = None,
) -> Matcher:
    """
    Match a Dockerfile instruction by type and properties.

    Examples:
        instruction(type="FROM", image_tag="latest")
        instruction(type="USER", user_name="root")
        instruction(type="ARG", arg_name_regex=r"(?i).*password.*")
    """
    params = {"instruction": type}

    # Add non-None parameters
    if base_image is not None:
        params["base_image"] = base_image
    if image_tag is not None:
        params["image_tag"] = image_tag
    if image_tag_regex is not None:
        params["image_tag_regex"] = image_tag_regex
    if missing_digest is not None:
        params["missing_digest"] = missing_digest
    if user_name is not None:
        params["user_name"] = user_name
    if user_name_regex is not None:
        params["user_name_regex"] = user_name_regex
    if port is not None:
        params["port"] = port
    if port_less_than is not None:
        params["port_less_than"] = port_less_than
    if port_greater_than is not None:
        params["port_greater_than"] = port_greater_than
    if protocol is not None:
        params["protocol"] = protocol
    if arg_name is not None:
        params["arg_name"] = arg_name
    if arg_name_regex is not None:
        params["arg_name_regex"] = arg_name_regex
    if copy_from is not None:
        params["copy_from"] = copy_from
    if chown is not None:
        params["chown"] = chown
    if missing_flag is not None:
        params["missing_flag"] = missing_flag
    if healthcheck_interval_less_than is not None:
        params["healthcheck_interval_less_than"] = healthcheck_interval_less_than
    if healthcheck_timeout_greater_than is not None:
        params["healthcheck_timeout_greater_than"] = healthcheck_timeout_greater_than
    if healthcheck_retries_greater_than is not None:
        params["healthcheck_retries_greater_than"] = healthcheck_retries_greater_than
    if label_key is not None:
        params["label_key"] = label_key
    if label_value_regex is not None:
        params["label_value_regex"] = label_value_regex
    if command_form is not None:
        params["command_form"] = command_form
    if workdir_not_absolute is not None:
        params["workdir_not_absolute"] = workdir_not_absolute
    if signal_not_in is not None:
        params["signal_not_in"] = signal_not_in
    if contains is not None:
        params["contains"] = contains
    if not_contains is not None:
        params["not_contains"] = not_contains
    if regex is not None:
        params["regex"] = regex
    if not_regex is not None:
        params["not_regex"] = not_regex
    if validate is not None:
        # Custom validation stored separately
        params["has_custom_validate"] = True

    return Matcher(type="instruction", params=params)


def missing(
    instruction: str,
    label_key: Optional[str] = None,
) -> Matcher:
    """
    Match when a Dockerfile instruction is missing.

    Examples:
        missing(instruction="USER")
        missing(instruction="HEALTHCHECK")
        missing(instruction="LABEL", label_key="maintainer")
    """
    params = {"instruction": instruction}
    if label_key is not None:
        params["label_key"] = label_key

    return Matcher(type="missing_instruction", params=params)


# --- docker-compose Matchers ---


def service_has(
    key: str,
    equals: Optional[Any] = None,
    not_equals: Optional[Any] = None,
    contains: Optional[str] = None,
    not_contains: Optional[str] = None,
    contains_any: Optional[List[str]] = None,
    regex: Optional[str] = None,
    env_name_regex: Optional[str] = None,
    env_value_regex: Optional[str] = None,
    volume_type: Optional[str] = None,
    source_regex: Optional[str] = None,
    target_regex: Optional[str] = None,
    published_port_less_than: Optional[int] = None,
) -> Matcher:
    """
    Match docker-compose services with specific properties.

    Examples:
        service_has(key="privileged", equals=True)
        service_has(key="volumes", contains="/var/run/docker.sock")
        service_has(key="network_mode", equals="host")
    """
    params = {"key": key}

    if equals is not None:
        params["equals"] = equals
    if not_equals is not None:
        params["not_equals"] = not_equals
    if contains is not None:
        params["contains"] = contains
    if not_contains is not None:
        params["not_contains"] = not_contains
    if contains_any is not None:
        params["contains_any"] = contains_any
    if regex is not None:
        params["regex"] = regex
    if env_name_regex is not None:
        params["env_name_regex"] = env_name_regex
    if env_value_regex is not None:
        params["env_value_regex"] = env_value_regex
    if volume_type is not None:
        params["volume_type"] = volume_type
    if source_regex is not None:
        params["source_regex"] = source_regex
    if target_regex is not None:
        params["target_regex"] = target_regex
    if published_port_less_than is not None:
        params["published_port_less_than"] = published_port_less_than

    return Matcher(type="service_has", params=params)


def service_missing(
    key: str,
    value_contains: Optional[str] = None,
) -> Matcher:
    """
    Match docker-compose services missing a property.

    Examples:
        service_missing(key="read_only")
        service_missing(key="security_opt")
        service_missing(key="security_opt", value_contains="no-new-privileges")
    """
    params = {"key": key}
    if value_contains is not None:
        params["value_contains"] = value_contains

    return Matcher(type="service_missing", params=params)
