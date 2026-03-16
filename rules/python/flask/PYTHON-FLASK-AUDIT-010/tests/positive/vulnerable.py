from flask import Flask, make_response
from flask_cors import CORS
from jinja2 import Environment
from markupsafe import Markup

app = Flask(__name__)

# Debug mode enabled
app.run(debug=True)

# Bind to all interfaces
app.run(host="0.0.0.0")

# CORS wildcard
CORS(app, origins="*")

# Cookie without secure flag
@app.route('/setcookie')
def setcookie():
    resp = make_response("cookie set")
    resp.set_cookie('session', 'value', secure=False, httponly=False)
    return resp

# Direct Jinja2 usage without autoescape
env = Environment(autoescape=False)

# Markup usage
html = Markup("<b>hello</b>")
