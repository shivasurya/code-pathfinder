from flask import Flask, request
import csv, io, requests, math

app = Flask(__name__)

def sanitize_csv_value(value):
    \"\"\"Strip leading formula characters from CSV values.\"\"\"
    if isinstance(value, str) and value and value[0] in ('=', '+', '-', '@'):
        return "'" + value  # Prefix with single quote to prevent formula execution
    return value

@app.route('/export')
def export_csv():
    name = request.args.get('name')
    writer = csv.writer(io.StringIO())
    # SAFE: Sanitize before writing to CSV
    writer.writerow([sanitize_csv_value(name), 'data'])

@app.route('/convert')
def convert():
    value = request.args.get('value')
    num = float(value)
    # SAFE: Reject NaN and Inf values
    if math.isnan(num) or math.isinf(num):
        return {'error': 'Invalid number'}, 400
    return {'result': num}

ALLOWED_HOSTS = {'api.example.com', 'cdn.example.com'}

@app.route('/proxy')
def proxy():
    host = request.args.get('host')
    # SAFE: Validate host against allowlist
    if host not in ALLOWED_HOSTS:
        return {'error': 'Host not allowed'}, 403
    response = requests.get(f'https://{host}/api/data')
    return response.text
