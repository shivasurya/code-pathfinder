from flask import Flask, request, make_response

app = Flask(__name__)

@app.route('/greet')
def greet():
    name = request.args.get('name')
    html = "<h1>Hello " + name + "</h1>"
    return html

@app.route('/profile')
def profile():
    bio = request.form.get('bio')
    resp = make_response("<div>" + bio + "</div>")
    return resp
