from django.core.mail import EmailMessage, send_mail


# SEC-060: XSS in email body
def vulnerable_email(request):
    body = request.POST.get('message')
    email = EmailMessage("Subject", body, "from@test.com", ["to@test.com"])
    email.content_subtype = "html"
    email.send()


# SEC-061: XSS in send_mail html_message
def vulnerable_sendmail(request):
    content = request.POST.get('body')
    send_mail("Subject", "text body", "from@test.com", ["to@test.com"],
              html_message=content)
