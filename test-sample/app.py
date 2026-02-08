"""Sample Python application for CI workflow validation."""


def greet(name: str) -> str:
    """Return a greeting message."""
    return f"Hello, {name}!"


def add(a: int, b: int) -> int:
    """Add two numbers."""
    return a + b


class Calculator:
    """Simple calculator."""

    def __init__(self):
        self.history = []

    def compute(self, a: int, b: int) -> int:
        result = a + b
        self.history.append(result)
        return result

    def get_history(self) -> list:
        return list(self.history)


if __name__ == "__main__":
    print(greet("World"))
    calc = Calculator()
    print(calc.compute(2, 3))
