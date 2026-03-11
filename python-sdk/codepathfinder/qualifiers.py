"""Value qualifiers for argument constraints in QueryType rules."""


class Qualifier:
    """Base class for value qualifiers used with .arg() constraints."""

    def to_constraint(self) -> dict:
        raise NotImplementedError


class LessThan(Qualifier):
    def __init__(self, value):
        self.value = value

    def to_constraint(self):
        return {"value": self.value, "wildcard": False, "comparator": "lt"}


class GreaterThan(Qualifier):
    def __init__(self, value):
        self.value = value

    def to_constraint(self):
        return {"value": self.value, "wildcard": False, "comparator": "gt"}


class LessThanOrEqual(Qualifier):
    def __init__(self, value):
        self.value = value

    def to_constraint(self):
        return {"value": self.value, "wildcard": False, "comparator": "lte"}


class GreaterThanOrEqual(Qualifier):
    def __init__(self, value):
        self.value = value

    def to_constraint(self):
        return {"value": self.value, "wildcard": False, "comparator": "gte"}


class Regex(Qualifier):
    def __init__(self, pattern):
        self.pattern = pattern

    def to_constraint(self):
        return {"value": self.pattern, "wildcard": False, "comparator": "regex"}


class Missing(Qualifier):
    def to_constraint(self):
        return {"value": None, "wildcard": False, "comparator": "missing"}


# Public API
def lt(n):
    """Less than numeric comparator."""
    return LessThan(n)


def gt(n):
    """Greater than numeric comparator."""
    return GreaterThan(n)


def lte(n):
    """Less than or equal numeric comparator."""
    return LessThanOrEqual(n)


def gte(n):
    """Greater than or equal numeric comparator."""
    return GreaterThanOrEqual(n)


def regex(pattern):
    """Regex match on string value."""
    return Regex(pattern)


def missing():
    """Keyword argument is absent from call."""
    return Missing()
