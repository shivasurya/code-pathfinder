from flask import Flask, request
import pickle
import yaml

app = Flask(__name__)

@app.route('/load', methods=['POST'])
def load_data():
    data = request.data
    obj = pickle.loads(data)
    return str(obj)

@app.route('/yaml_load', methods=['POST'])
def yaml_load_data():
    raw = request.data
    obj = yaml.load(raw)
    return str(obj)
