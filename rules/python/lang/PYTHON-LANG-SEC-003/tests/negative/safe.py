# Use ast.literal_eval for safe literal parsing
import ast
result = ast.literal_eval(user_expr)

# Whitelist allowed modules for dynamic imports
ALLOWED_MODULES = {"json", "math", "datetime"}
if module_name in ALLOWED_MODULES:
    mod = __import__(module_name)
