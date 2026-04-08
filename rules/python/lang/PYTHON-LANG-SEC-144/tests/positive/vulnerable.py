from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from flask_cors import CORS

app = FastAPI()

# SEC-144: CORS wildcard with CORSMiddleware constructor
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# SEC-144: CORSMiddleware called directly
middleware = CORSMiddleware(app, allow_origins=["*"])

# SEC-144: Flask-CORS wildcard
flask_app = Flask(__name__)
CORS(flask_app, origins="*")

# SEC-144: CORS with allow_origin_regex wildcard
app.add_middleware(
    CORSMiddleware,
    allow_origin_regex=".*",
    allow_credentials=True,
)
