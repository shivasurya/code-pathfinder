from flask import Flask
from hashids import Hashids

app = Flask(__name__)
app.secret_key = 'my-secret'
hasher = Hashids(salt=app.secret_key)
