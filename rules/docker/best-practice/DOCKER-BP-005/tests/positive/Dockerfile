FROM ubuntu:22.04

# Bad: Installs nginx + 50+ recommended packages
RUN apt-get update && \
    apt-get install -y nginx curl

# This pulls in:
# - X11 libraries (for desktop apps, not needed in containers)
# - Documentation packages
# - Example/demo files
# - Optional utilities
