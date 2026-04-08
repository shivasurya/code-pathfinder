from flask import render_template
from jinja2 import Environment, FileSystemLoader

# Safe: render_template with file-based template and data passed as context
def render_profile(user):
    return render_template("profile.html", user=user)

# Safe: Loading templates from filesystem via get_template
def render_invoice(invoice_data):
    env = Environment(loader=FileSystemLoader("templates"))
    tmpl = env.get_template("invoice.html")
    return tmpl.render(invoice=invoice_data)

# Safe: Using get_template from a Jinja2 Environment (file-based)
def render_static_header():
    from jinja2 import Environment, FileSystemLoader
    env = Environment(loader=FileSystemLoader("templates"))
    tmpl = env.get_template("header.html")
    return tmpl.render()

# Safe: select_template from predefined list
def render_themed_page(theme, data):
    env = Environment(loader=FileSystemLoader("templates"))
    tmpl = env.select_template([f"{theme}.html", "default.html"])
    return tmpl.render(data=data)

# Safe: Using Markup for escaping (not template construction)
from markupsafe import Markup
def safe_output(text):
    return Markup.escape(text)
