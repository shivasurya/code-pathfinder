# --- file: app.py ---
from flask import Flask, request
from renderer import render_greeting

app = Flask(__name__)


@app.route('/greet')
def greet():
    name = request.args.get('name')
    html = render_greeting(name)
    return html

# --- file: renderer.py ---
from flask import render_template_string


def render_greeting(username):
    template = "<h1>Hello " + username + "</h1>"
    return render_template_string(template)
