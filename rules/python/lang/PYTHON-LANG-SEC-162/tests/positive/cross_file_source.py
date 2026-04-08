# SEC-162: Source file — path constructed from user input

import os
from flask import Flask, request
from cross_file_sink import process_repo_file, read_translation_file

app = Flask(__name__)


@app.route('/repo/file')
def repo_file_endpoint():
    repo_slug = request.args.get("repo")
    filename = request.args.get("file")
    path = os.path.join("/srv/repos", repo_slug, filename)
    content = process_repo_file(path)
    return content


@app.route('/translation/download')
def download_translation():
    lang = request.args.get("lang")
    path = os.path.join("/srv/translations", lang)
    data = read_translation_file(path)
    return data
