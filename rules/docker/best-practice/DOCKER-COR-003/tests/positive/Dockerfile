FROM ubuntu:22.04

# Bad: Multiple CMDs - only last one is used
CMD ["echo", "first"]       # Ignored silently
CMD ["python3", "app.py"]   # Ignored silently
CMD ["nginx", "-g", "daemon off;"]  # Only this one takes effect
