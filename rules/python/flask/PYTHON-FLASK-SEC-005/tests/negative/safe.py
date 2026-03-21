from flask import Flask, request
import ast
import json

app = Flask(__name__)

@app.route('/calculate')
def calculate():
    expression = request.args.get('expr')
    # SAFE: ast.literal_eval only evaluates literals (strings, numbers, tuples, etc.)
    try:
        result = ast.literal_eval(expression)
    except (ValueError, SyntaxError):
        return {'error': 'Invalid expression'}, 400
    return {'result': result}

# For math expressions, use a dedicated safe parser
# For structured data, use json.loads()
