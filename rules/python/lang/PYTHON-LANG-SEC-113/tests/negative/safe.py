import os
from flask import request

# Safe: Using os.environ.get for server-side environment variables
def get_config():
    db_host = os.environ.get("DATABASE_HOST")
    return db_host

# Safe: Using REMOTE_ADDR (server-determined, not attacker-controlled)
def get_client_ip():
    addr = request.environ.get("REMOTE_ADDR")
    return addr

# Safe: Using request.remote_addr property
def log_access():
    ip = request.remote_addr
    print(f"Access from {ip}")

# Safe: Using SERVER_NAME (server config, not from request)
def get_server_name():
    name = request.environ.get("SERVER_NAME")
    return name

# Safe: Using os.environ for deployment config
def is_production():
    env = os.environ.get("FLASK_ENV", "development")
    return env == "production"
