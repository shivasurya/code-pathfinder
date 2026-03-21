from flask import Flask, request
import jwt

app = Flask(__name__)

@app.route('/token')
def create_token():
    # Vulnerable: user input flows directly into JWT payload
    user_data = request.args.get('user')
    return jwt.encode({"sub": user_data}, "key", algorithm="HS256")
