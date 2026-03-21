import socket, paramiko
# Bind to localhost only for internal services
sock = socket.socket()
sock.bind(("127.0.0.1", 8080))
# Verify SSH host keys explicitly
client = paramiko.SSHClient()
client.load_system_host_keys()
client.set_missing_host_key_policy(paramiko.RejectPolicy())
client.connect(host)
# Validate commands before remote execution
ALLOWED_COMMANDS = {"ls", "df", "uptime"}
if cmd not in ALLOWED_COMMANDS:
    raise ValueError("Unauthorized command")
client.exec_command(cmd)
# Use authenticated Pipe() for multiprocessing
parent_conn, child_conn = multiprocessing.Pipe()
