from flask import Flask, request

app = Flask(__name__)

@app.route('/read')
def read_file():
    filename = request.args.get('file')
    content = open(filename, 'r').read()
    return content
