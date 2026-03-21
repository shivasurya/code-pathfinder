from flask import Flask, request

app = Flask(__name__)

@app.route('/echo')
def echo():
    msg = request.args.get('msg')
    return "<p>" + msg + "</p>"
