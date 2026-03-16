from flask import Flask, request

app = Flask(__name__)

@app.route('/calc')
def calculator():
    expr = request.args.get('expr')
    result = eval(expr)
    return str(result)

@app.route('/run_code')
def run_code():
    code = request.form.get('code')
    exec(code)
    return "executed"
