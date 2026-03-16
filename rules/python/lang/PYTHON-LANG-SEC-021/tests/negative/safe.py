import subprocess
# Pass arguments as a list (no shell interpretation)
subprocess.run(["grep", user_input, "log"], check=True)
# Use asyncio exec variant (no shell)
await asyncio.create_subprocess_exec("grep", user_input, "log")
# Validate and restrict allowed commands
ALLOWED_COMMANDS = {"ls", "grep", "wc"}
if cmd not in ALLOWED_COMMANDS:
    raise ValueError("Command not allowed")
