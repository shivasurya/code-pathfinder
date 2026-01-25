"""Comprehensive Python file covering all 12 symbol types."""

from typing import Protocol
from enum import Enum, IntEnum, Flag
from dataclasses import dataclass
from abc import ABC

# ===== CONSTANTS (UPPERCASE module variables) =====
MAX_CONNECTIONS = 100
API_KEY = "secret_key"
DEFAULT_TIMEOUT = 30


# ===== MODULE VARIABLES (lowercase module variables) =====
version = "1.0.0"
debug_mode = False


# ===== FUNCTION_DEFINITION (module-level functions) =====
def module_level_function(x, y):
    """A regular module-level function."""
    return x + y


def another_function():
    """Another module-level function."""
    pass


# ===== INTERFACE (Protocol/ABC classes) =====
class Drawable(Protocol):
    """Protocol/Interface definition."""

    def draw(self) -> None:
        """Draw method."""
        pass


class Storage(ABC):
    """ABC-based interface."""

    def save(self, data):
        """Save data."""
        pass


# ===== ENUM (Enum classes) =====
class Color(Enum):
    """Color enumeration."""
    RED = 1
    GREEN = 2
    BLUE = 3


class Priority(IntEnum):
    """Priority enumeration."""
    LOW = 1
    MEDIUM = 2
    HIGH = 3


class Flags(Flag):
    """Flag enumeration."""
    READ = 1
    WRITE = 2
    EXECUTE = 4


# ===== DATACLASS (@dataclass classes) =====
@dataclass
class Point:
    """A simple point dataclass."""
    x: int
    y: int


@dataclass
class Rectangle:
    """Rectangle dataclass with a method."""
    width: int
    height: int

    def area(self):
        """Calculate area (method in dataclass)."""
        return self.width * self.height


# ===== REGULAR CLASS with all method types =====
class ComprehensiveClass:
    """Class demonstrating all method types."""

    # CLASS_FIELD (class-level attributes)
    class_variable = "shared"
    MAX_SIZE = 1000

    # CONSTRUCTOR (__init__ method)
    def __init__(self, value, name):
        """Constructor."""
        self.value = value
        self.name = name

    # METHOD (regular instance method)
    def regular_method(self):
        """Regular method."""
        return self.value

    def another_method(self, x):
        """Another regular method."""
        return self.value + x

    # PROPERTY (@property decorator)
    @property
    def name_property(self):
        """Property method."""
        return self.name

    @property
    def value_property(self):
        """Another property."""
        return self.value

    # SPECIAL_METHOD (magic methods)
    def __str__(self):
        """Special method: string representation."""
        return f"ComprehensiveClass({self.value})"

    def __repr__(self):
        """Special method: repr."""
        return f"ComprehensiveClass(value={self.value}, name={self.name})"

    def __add__(self, other):
        """Special method: addition."""
        return ComprehensiveClass(self.value + other.value, self.name)

    def __len__(self):
        """Special method: length."""
        return len(str(self.value))

    def __eq__(self, other):
        """Special method: equality."""
        return self.value == other.value

    def __call__(self):
        """Special method: callable."""
        return self.value

    def __getitem__(self, key):
        """Special method: indexing."""
        return self.value

    def __contains__(self, item):
        """Special method: contains."""
        return item in str(self.value)
