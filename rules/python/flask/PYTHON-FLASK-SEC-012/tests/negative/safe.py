from flask import Flask, request, redirect, url_for
from urllib.parse import urlparse

app = Flask(__name__)

def is_safe_redirect_url(target):
    \"\"\"Verify the redirect URL is relative or points to the same host.\"\"\"
    host_url = urlparse(request.host_url)
    redirect_url = urlparse(target)
    return (redirect_url.scheme in ('', 'http', 'https') and
            redirect_url.netloc in ('', host_url.netloc))

@app.route('/login', methods=['POST'])
def login():
    # ... authenticate user ...
    next_url = request.args.get('next')
    # SAFE: Validate redirect URL before use
    if next_url and is_safe_redirect_url(next_url):
        return redirect(next_url)
    return redirect(url_for('index'))  # Default to known safe route
