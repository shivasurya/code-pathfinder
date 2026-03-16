import xml.etree.ElementTree as ET
import xml.dom.minidom
import xml.sax
import xmlrpc.client
import csv

# SEC-090: insecure XML parsing
tree = ET.parse("data.xml")
root = ET.fromstring("<root/>")

# SEC-091: minidom
doc = xml.dom.minidom.parse("data.xml")
doc2 = xml.dom.minidom.parseString("<root/>")

# SEC-092: xmlrpc
proxy = xmlrpc.client.ServerProxy("http://example.com/rpc")

# SEC-093: mako
from mako.template import Template
tmpl = Template("Hello ${name}")

# SEC-094: csv.writer
import io
writer = csv.writer(io.StringIO())
dict_writer = csv.DictWriter(io.StringIO(), fieldnames=["a", "b"])
