import xml.etree.ElementTree as ET
import xml.dom.minidom
import xml.sax
import xmlrpc.client
import csv

# SEC-091: minidom
doc = xml.dom.minidom.parse("data.xml")
doc2 = xml.dom.minidom.parseString("<root/>")
