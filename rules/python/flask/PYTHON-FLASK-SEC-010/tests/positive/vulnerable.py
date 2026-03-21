from flask import Flask, request

app = Flask(__name__)

@app.route('/convert')
def convert():
    value = request.args.get('value')
    result = float(value)
    return str(result)
