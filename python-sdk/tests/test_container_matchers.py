"""Tests for container matchers."""

from rules.container_matchers import (
    instruction,
    missing,
    service_has,
    service_missing,
)


class TestInstructionMatcher:
    def test_basic_type(self):
        m = instruction(type="FROM")
        d = m.to_dict()
        assert d["type"] == "instruction"
        assert d["instruction"] == "FROM"

    def test_with_properties(self):
        m = instruction(type="FROM", image_tag="latest")
        d = m.to_dict()
        assert d["image_tag"] == "latest"

    def test_user_instruction(self):
        m = instruction(type="USER", user_name="root")
        d = m.to_dict()
        assert d["instruction"] == "USER"
        assert d["user_name"] == "root"

    def test_user_instruction_regex(self):
        m = instruction(type="USER", user_name_regex=r"^root$")
        d = m.to_dict()
        assert d["instruction"] == "USER"
        assert d["user_name_regex"] == r"^root$"

    def test_arg_with_regex(self):
        m = instruction(type="ARG", arg_name_regex=r"(?i).*password.*")
        d = m.to_dict()
        assert d["arg_name_regex"] == r"(?i).*password.*"

    def test_expose_port_range(self):
        m = instruction(type="EXPOSE", port_less_than=1024)
        d = m.to_dict()
        assert d["port_less_than"] == 1024

    def test_generic_contains(self):
        m = instruction(type="RUN", contains="sudo")
        d = m.to_dict()
        assert d["contains"] == "sudo"

    def test_all_from_params(self):
        m = instruction(
            type="FROM",
            base_image="ubuntu",
            image_tag="latest",
            image_tag_regex=r".*latest.*",
            missing_digest=True,
        )
        d = m.to_dict()
        assert d["base_image"] == "ubuntu"
        assert d["image_tag"] == "latest"
        assert d["image_tag_regex"] == r".*latest.*"
        assert d["missing_digest"] is True

    def test_all_expose_params(self):
        m = instruction(
            type="EXPOSE",
            port=80,
            port_less_than=1024,
            port_greater_than=1023,
            protocol="tcp",
        )
        d = m.to_dict()
        assert d["port"] == 80
        assert d["port_less_than"] == 1024
        assert d["port_greater_than"] == 1023
        assert d["protocol"] == "tcp"

    def test_arg_params(self):
        m = instruction(
            type="ARG", arg_name="VERSION", arg_name_regex=r"(?i).*password.*"
        )
        d = m.to_dict()
        assert d["arg_name"] == "VERSION"
        assert d["arg_name_regex"] == r"(?i).*password.*"

    def test_copy_params(self):
        m = instruction(
            type="COPY", copy_from="builder", chown="user:group", missing_flag="--chown"
        )
        d = m.to_dict()
        assert d["copy_from"] == "builder"
        assert d["chown"] == "user:group"
        assert d["missing_flag"] == "--chown"

    def test_healthcheck_params(self):
        m = instruction(
            type="HEALTHCHECK",
            healthcheck_interval_less_than="30s",
            healthcheck_timeout_greater_than="10s",
            healthcheck_retries_greater_than=3,
        )
        d = m.to_dict()
        assert d["healthcheck_interval_less_than"] == "30s"
        assert d["healthcheck_timeout_greater_than"] == "10s"
        assert d["healthcheck_retries_greater_than"] == 3

    def test_label_params(self):
        m = instruction(
            type="LABEL", label_key="maintainer", label_value_regex=r".*@example\.com"
        )
        d = m.to_dict()
        assert d["label_key"] == "maintainer"
        assert d["label_value_regex"] == r".*@example\.com"

    def test_cmd_params(self):
        m = instruction(type="CMD", command_form="shell")
        d = m.to_dict()
        assert d["command_form"] == "shell"

    def test_workdir_params(self):
        m = instruction(type="WORKDIR", workdir_not_absolute=True)
        d = m.to_dict()
        assert d["workdir_not_absolute"] is True

    def test_stopsignal_params(self):
        m = instruction(type="STOPSIGNAL", signal_not_in=["SIGTERM", "SIGKILL"])
        d = m.to_dict()
        assert d["signal_not_in"] == ["SIGTERM", "SIGKILL"]

    def test_generic_matchers(self):
        m = instruction(
            type="RUN",
            contains="sudo",
            not_contains="rm -rf",
            regex=r".*apt-get.*",
            not_regex=r".*yum.*",
        )
        d = m.to_dict()
        assert d["contains"] == "sudo"
        assert d["not_contains"] == "rm -rf"
        assert d["regex"] == r".*apt-get.*"
        assert d["not_regex"] == r".*yum.*"

    def test_custom_validate(self):
        m = instruction(type="RUN", validate=lambda x: True)
        d = m.to_dict()
        assert d["has_custom_validate"] is True


class TestMissingMatcher:
    def test_missing_instruction(self):
        m = missing(instruction="USER")
        d = m.to_dict()
        assert d["type"] == "missing_instruction"
        assert d["instruction"] == "USER"

    def test_missing_label(self):
        m = missing(instruction="LABEL", label_key="maintainer")
        d = m.to_dict()
        assert d["instruction"] == "LABEL"
        assert d["label_key"] == "maintainer"


class TestServiceHasMatcher:
    def test_equals(self):
        m = service_has(key="privileged", equals=True)
        d = m.to_dict()
        assert d["type"] == "service_has"
        assert d["key"] == "privileged"
        assert d["equals"] is True

    def test_contains_any(self):
        m = service_has(
            key="volumes", contains_any=["/var/run/docker.sock", "/run/docker.sock"]
        )
        d = m.to_dict()
        assert len(d["contains_any"]) == 2

    def test_env_regex(self):
        m = service_has(key="environment", env_name_regex=r"(?i).*password.*")
        d = m.to_dict()
        assert d["env_name_regex"] == r"(?i).*password.*"

    def test_all_params(self):
        m = service_has(
            key="volumes",
            equals="/data",
            not_equals="/tmp",
            contains="docker.sock",
            not_contains="random",
            contains_any=["/var/run", "/run"],
            regex=r".*/docker\.sock",
            env_name_regex=r"(?i)password",
            env_value_regex=r"(?i)secret",
            volume_type="bind",
            source_regex=r"^/host",
            target_regex=r"^/container",
            published_port_less_than=1024,
        )
        d = m.to_dict()
        assert d["equals"] == "/data"
        assert d["not_equals"] == "/tmp"
        assert d["contains"] == "docker.sock"
        assert d["not_contains"] == "random"
        assert d["contains_any"] == ["/var/run", "/run"]
        assert d["regex"] == r".*/docker\.sock"
        assert d["env_name_regex"] == r"(?i)password"
        assert d["env_value_regex"] == r"(?i)secret"
        assert d["volume_type"] == "bind"
        assert d["source_regex"] == r"^/host"
        assert d["target_regex"] == r"^/container"
        assert d["published_port_less_than"] == 1024


class TestServiceMissingMatcher:
    def test_missing_key(self):
        m = service_missing(key="read_only")
        d = m.to_dict()
        assert d["type"] == "service_missing"
        assert d["key"] == "read_only"

    def test_missing_with_value(self):
        m = service_missing(key="security_opt", value_contains="no-new-privileges")
        d = m.to_dict()
        assert d["value_contains"] == "no-new-privileges"
