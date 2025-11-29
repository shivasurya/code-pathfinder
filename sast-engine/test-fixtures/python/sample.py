"""Sample Python file for testing parser"""


class Calculator:
    """A simple calculator class"""
    
    def __init__(self):
        self.result = 0
    
    def add(self, x, y):
        """Add two numbers"""
        result = x + y
        return result
    
    def subtract(self, x, y):
        """Subtract two numbers"""
        assert y != 0, "Cannot divide by zero"
        return x - y
    
    def multiply(self, x, y):
        """Multiply two numbers"""
        for i in range(y):
            if i == 10:
                break
            self.result += x
        return self.result


def fibonacci(n):
    """Generate fibonacci sequence"""
    a, b = 0, 1
    for _ in range(n):
        yield a
        a, b = b, a + b


def process_data(data):
    """Process data with error handling"""
    if not data:
        return None
    
    processed = []
    for item in data:
        if item < 0:
            continue
        processed.append(item * 2)
    
    return processed


def main():
    """Main function"""
    calc = Calculator()
    result = calc.add(10, 20)
    print(f"Result: {result}")
    
    for num in fibonacci(10):
        print(num)


if __name__ == "__main__":
    main()
