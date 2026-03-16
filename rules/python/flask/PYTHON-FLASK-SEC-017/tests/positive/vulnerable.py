from flask import Flask, request, send_from_directory

app = Flask(__name__)

@app.route('/files')
def serve_file():
    filename = request.args.get('file')
    return send_from_directory('/uploads', filename)
