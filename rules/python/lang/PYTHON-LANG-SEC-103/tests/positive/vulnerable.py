import uuid
import os
import re
import logging
import logging.config

# SEC-103: regex DoS
pattern = re.compile(r"(a+)+$")
re.match(r"(a|b)*c", user_input)
re.search(r"(\d+\.)+", text)
