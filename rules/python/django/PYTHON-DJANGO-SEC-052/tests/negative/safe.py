from django.http import HttpResponse
from django.utils.html import escape, strip_tags
from django.template import loader

def greet_user(request):
    # SECURE: Use Django template with auto-escaping
    template = loader.get_template('greet.html')
    name = request.GET.get('name', '')
    return HttpResponse(template.render({'name': name}))
    # Django templates auto-escape {{ name }} by default

def greet_user_direct(request):
    # SECURE: Explicitly escape user input in HttpResponse
    name = escape(request.GET.get('name', ''))
    return HttpResponse("<h1>Hello, " + name + "!</h1>")

def render_comment(request):
    # SECURE: Sanitize before marking as safe, or use template auto-escaping
    comment = request.POST.get('comment', '')
    # Option 1: Strip all HTML tags
    clean_comment = strip_tags(comment)
    # Option 2: Use bleach to allow safe HTML subset
    # clean_comment = bleach.clean(comment, tags=['b', 'i', 'em', 'strong'])
    return render(request, 'comment.html', {'comment': clean_comment})
