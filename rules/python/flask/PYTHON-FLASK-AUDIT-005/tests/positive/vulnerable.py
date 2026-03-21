from flask import Flask, url_for

app = Flask(__name__)

@app.route('/link')
def get_link():
    return url_for('index', _external=True)
