# Test file with various function calls

def process_data(data):
    """Process data with various function calls."""
    # Simple function calls
    sanitize(data)
    validate(data)

    # Method calls
    db.query(data)
    logger.info("Processing")

    # Calls with arguments
    transform(data, mode="strict")
    calculate(x, y, precision=2)

    # Nested calls
    result = sanitize(validate(data))

    return result

def helper_function():
    """Helper with self-calls."""
    process_data(get_data())

class DataProcessor:
    def process(self):
        """Method with calls."""
        self.validate()
        self.db.execute()
        external.function()
