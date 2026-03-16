import xml.etree.ElementTree as ET
import xml.dom.minidom
import xml.sax
import xmlrpc.client
import csv

# SEC-094: csv.writer
import io
writer = csv.writer(io.StringIO())
dict_writer = csv.DictWriter(io.StringIO(), fieldnames=["a", "b"])
