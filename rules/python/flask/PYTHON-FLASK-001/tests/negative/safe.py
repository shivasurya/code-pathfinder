import os
from flask import Flask, request

app = Flask(__name__)

@app.route('/api/users')
def get_users():
    return {'users': [...]}

if __name__ == '__main__':
    # SAFE: Debug mode explicitly disabled
    app.run(debug=False)

    # BETTER: Use environment variable
    debug_mode = os.getenv('FLASK_DEBUG', 'False') == 'True'
    app.run(debug=debug_mode)

    # BEST: Don't set it at all (defaults to False)
    app.run()  # debug=False is the default
