from flask import Flask, request, abort
from werkzeug.utils import secure_filename
import os

app = Flask(__name__)
UPLOAD_DIR = '/uploads'

@app.route('/download')
def download_file():
    filename = request.args.get('file')
    # SAFE: secure_filename strips directory separators and traversal sequences
    safe_name = secure_filename(filename)
    filepath = os.path.join(UPLOAD_DIR, safe_name)

    # Additional check: ensure resolved path is within allowed directory
    if not os.path.realpath(filepath).startswith(os.path.realpath(UPLOAD_DIR)):
        abort(403)

    with open(filepath, 'r') as f:
        return f.read()
