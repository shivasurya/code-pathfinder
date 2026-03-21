from flask import Flask, make_response

app = Flask(__name__)

@app.route('/setcookie')
def setcookie():
    resp = make_response("cookie set")
    resp.set_cookie('session', 'value', secure=False, httponly=False)
    return resp
