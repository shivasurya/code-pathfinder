from flask import Flask, request, render_template
from markupsafe import escape

app = Flask(__name__)

@app.route('/greet')
def greet():
    name = request.args.get('name')
    # SAFE Option 1: Use Jinja2 template (auto-escapes by default)
    return render_template('greet.html', name=name)

@app.route('/greet-inline')
def greet_inline():
    name = request.args.get('name')
    # SAFE Option 2: Explicitly escape user input
    safe_name = escape(name)
    return f'<h1>Hello, {safe_name}!</h1>'
