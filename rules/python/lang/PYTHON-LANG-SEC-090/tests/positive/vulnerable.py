import xml.etree.ElementTree as ET
import xml.dom.minidom
import xml.sax
import xmlrpc.client
import csv

# SEC-090: insecure XML parsing
tree = ET.parse("data.xml")
root = ET.fromstring("<root/>")
