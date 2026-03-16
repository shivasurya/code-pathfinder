import uuid
import os
import re
import logging
import logging.config

# SEC-101: insecure file permissions
os.chmod("/tmp/data", 0o777)
os.fchmod(3, 0o666)
