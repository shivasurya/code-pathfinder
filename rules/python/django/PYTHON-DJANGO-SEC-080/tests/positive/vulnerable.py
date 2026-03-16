from django.contrib.auth.models import User


# SEC-080: set_password with empty string
def reset_password_empty(user):
    user.set_password("")
    user.save()


# SEC-081: POST data flowing to set_password
def change_password(request):
    password = request.POST.get('password')
    user = User.objects.get(id=1)
    user.set_password(password)
    user.save()
