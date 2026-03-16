import subprocess
import shlex
# Use subprocess with list arguments (no shell interpretation)
subprocess.run(["ls", user_input], check=True)
# Or properly quote shell arguments
subprocess.run(f"ls {shlex.quote(user_input)}", shell=True, check=True)
