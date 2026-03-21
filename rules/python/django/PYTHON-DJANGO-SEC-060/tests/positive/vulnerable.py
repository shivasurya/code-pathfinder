from django.core.mail import EmailMessage, send_mail

    body = request.POST.get('message')
    email = EmailMessage("Subject", body, "from@test.com", ["to@test.com"])
    email.content_subtype = "html"
    email.send()


    content = request.POST.get('body')
    send_mail("Subject", "text body", "from@test.com", ["to@test.com"],
              html_message=content)
