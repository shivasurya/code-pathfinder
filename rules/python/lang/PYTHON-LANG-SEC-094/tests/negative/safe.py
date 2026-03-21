# Use defusedxml for safe XML parsing
import defusedxml.ElementTree as ET
tree = ET.parse(user_uploaded_xml)  # XXE protection enabled
import defusedxml.minidom
doc = defusedxml.minidom.parseString(user_xml_string)
# Restrict Mako templates to trusted sources only
from mako.template import Template
t = Template(filename="/app/templates/report.html")  # Trusted file, not user input
# Sanitize CSV cell values to prevent formula injection
def sanitize_csv_value(value):
    if isinstance(value, str) and value and value[0] in "=+-@":
        return "'" + value  # Prefix with single quote
    return value
