from flask import Flask, request

app = Flask(__name__)

@app.route('/api/users')
def get_users():
    # Some application logic
    return {'users': [...]}

if __name__ == '__main__':
    # DANGEROUS: Debug mode enabled
    app.run(debug=True)  # Vulnerable!

# Also vulnerable:
# app.debug = True
# app.run()

# Or via config:
# app.config['DEBUG'] = True
