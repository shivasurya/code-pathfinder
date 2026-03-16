from flask import Flask, request
import csv, io

app = Flask(__name__)

@app.route('/export')
def export_csv():
    name = request.args.get('name')
    output = io.StringIO()
    writer = csv.writer(output)
    writer.writerow([name, "data"])
    return output.getvalue()
