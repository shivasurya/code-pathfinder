import uuid
import os
import re
import logging
import logging.config

# SEC-100: uuid1 (leaks MAC)
uid = uuid.uuid1()

# SEC-101: insecure file permissions
os.chmod("/tmp/data", 0o777)
os.fchmod(3, 0o666)

# SEC-102: hardcoded password
import mysql.connector
conn = mysql.connector.connect(host="db", password="secret123")

# SEC-103: regex DoS
pattern = re.compile(r"(a+)+$")
re.match(r"(a|b)*c", user_input)
re.search(r"(\d+\.)+", text)

# SEC-104: logging.config.listen
server = logging.config.listen(9999)

# SEC-105: logger credential leak
logging.info("User logged in with password: %s", password)
logging.debug("API key: %s", api_key)
