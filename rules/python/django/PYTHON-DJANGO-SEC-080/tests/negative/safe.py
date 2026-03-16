from django.contrib.auth.models import User
from django.core.exceptions import ValidationError

def reset_password(request):
    # SECURE: Validate password is present and meets requirements
    password = request.POST.get('password')
    if not password or len(password) < 8:
        raise ValidationError("Password must be at least 8 characters")
    user = User.objects.get(id=request.user.id)
    user.set_password(password)
    user.save()

def create_user_no_password(request):
    # SECURE: Use set_unusable_password() instead of empty string
    user = User.objects.create_user(
        username=request.POST.get('username'),
    )
    user.set_unusable_password()  # Correct way to create account without password
    user.save()

def create_user_with_validation(request):
    # SECURE: Use Django's password validators
    from django.contrib.auth.password_validation import validate_password
    password = request.POST.get('password')
    validate_password(password)  # Raises ValidationError if too weak
    user = User.objects.create_user(
        username=request.POST.get('username'),
        password=password,
    )
