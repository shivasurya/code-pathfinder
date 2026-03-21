from pyramid.config import Configurator
from pyramid.response import Response
from markupsafe import escape
from sqlalchemy import text

# SECURE: CSRF protection enabled (default)
config = Configurator()
config.set_default_csrf_options(require_csrf=True, token='csrf_token')

# SECURE: Escape user input or use templates with auto-escaping
@view_config(route_name='greet', renderer='templates/greet.jinja2')
def greet(request):
    name = request.params.get('name', '')
    return {'name': name}  # Jinja2 auto-escapes by default

# Or if using Response directly:
@view_config(route_name='greet')
def greet_safe(request):
    name = escape(request.params.get('name', ''))
    return Response(f'<h1>Hello, {name}!</h1>')

# SECURE: Parameterized SQL queries
@view_config(route_name='search')
def search(request):
    query = request.params.get('q', '')
    results = DBSession.query(User).filter(
        User.name.like(f'%{query}%')  # ORM method - safe
    ).all()
    # Or with raw SQL using bindparams:
    stmt = text("SELECT * FROM users WHERE name LIKE :q").bindparams(q=f'%{query}%')
    return {'results': results}
