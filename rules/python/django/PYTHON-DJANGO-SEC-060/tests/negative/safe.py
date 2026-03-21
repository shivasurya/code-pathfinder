from django.core.mail import send_mail
from django.utils.html import escape, strip_tags
from django.template.loader import render_to_string

def send_welcome(request):
    # SECURE: Escape user input before including in HTML email
    name = escape(request.POST.get('name', ''))
    html_body = render_to_string('emails/welcome.html', {'name': name})
    send_mail(
        'Welcome!',
        strip_tags(html_body),  # Plain text fallback
        'noreply@example.com',
        [request.POST.get('email')],
        html_message=html_body,
    )

def send_notification(request):
    # SECURE: Use Django template for HTML email with auto-escaping
    message = request.POST.get('message', '')
    context = {'message': message}  # Auto-escaped in template
    html_content = render_to_string('emails/notification.html', context)
    send_mail(
        'Notification',
        strip_tags(html_content),
        'noreply@example.com',
        ['admin@example.com'],
        html_message=html_content,
    )
