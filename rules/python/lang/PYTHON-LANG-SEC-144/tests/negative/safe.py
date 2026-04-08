from fastapi import FastAPI

app = FastAPI()

# Safe: no CORS middleware configured at all
# The rule detects CORSMiddleware(), *.add_middleware(), and CORS() calls.
# These patterns do not invoke any of those.

# Safe: app with no CORS - just basic routes
app_no_cors = FastAPI()

# Safe: using security headers via response directly (not middleware call)
from starlette.responses import JSONResponse

async def homepage():
    return JSONResponse({"status": "ok"})

# Safe: configuring allowed hosts in settings (no middleware call)
ALLOWED_ORIGINS = [
    "https://example.com",
    "https://app.example.com",
]

# Safe: just storing config, not calling add_middleware
cors_config = {
    "allow_origins": ALLOWED_ORIGINS,
    "allow_credentials": True,
}
