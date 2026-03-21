from flask import Flask, request, redirect

app = Flask(__name__)

@app.route('/goto')
def goto():
    url = request.args.get('url')
    return redirect(url)

@app.route('/redir')
def redir():
    next_page = request.form.get('next')
    return redirect(next_page)
