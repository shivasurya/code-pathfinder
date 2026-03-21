from flask import Flask, request, render_template, render_template_string

app = Flask(__name__)

@app.route('/hello')
def hello():
    name = request.args.get('name', 'World')
    # SAFE Option 1: Use render_template with a file-based template
    return render_template('hello.html', name=name)

@app.route('/hello-inline')
def hello_inline():
    name = request.args.get('name', 'World')
    # SAFE Option 2: Pass user input as a template variable, never in the template string
    return render_template_string('<h1>Hello, {{ name }}!</h1>', name=name)
