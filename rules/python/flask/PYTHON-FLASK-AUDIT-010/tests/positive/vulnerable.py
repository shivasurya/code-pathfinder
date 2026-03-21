from flask import Flask

app = Flask(__name__)
app.config['WTF_CSRF_ENABLED'] = False
