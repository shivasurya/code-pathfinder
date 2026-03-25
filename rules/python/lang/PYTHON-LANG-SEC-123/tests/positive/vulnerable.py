from flask import request, render_template_string
from jinja2 import Environment, Template, SandboxedEnvironment

# SEC-123: Jinja2 SSTI via template construction from dynamic input

# 1. render_template_string with user input
def render_greeting():
    name = request.args.get("name")
    template = f"<h1>Hello {name}</h1>"
    return render_template_string(template)

# 2. Environment.from_string with user-controlled template
def render_custom_template():
    env = Environment()
    user_template = request.form.get("template")
    tmpl = env.from_string(user_template)
    return tmpl.render()

# 3. Direct Template constructor with dynamic content
def render_email_body(template_body):
    tmpl = Template(template_body)
    return tmpl.render(user="admin")

# 4. SandboxedEnvironment.from_string (sandbox can be escaped)
def render_sandboxed(user_input):
    env = SandboxedEnvironment()
    tmpl = env.from_string(user_input)
    return tmpl.render()

# 5. Template from database or external source
def render_stored_template(db):
    template_str = db.query("SELECT template FROM email_templates WHERE id = 1").scalar()
    tmpl = Template(template_str)
    return tmpl.render(site_name="Example")

# 6. Jinja2 Template in error handler
def custom_error_page(error_message):
    tmpl = Template("<div class='error'>{{ error }}</div>")
    return render_template_string(f"<html><body>{error_message}</body></html>")
