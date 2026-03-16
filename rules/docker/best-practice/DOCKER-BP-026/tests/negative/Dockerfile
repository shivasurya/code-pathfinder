FROM fedora:39

# Good: Using -y flag for non-interactive installation
RUN dnf install -y nginx && \
    dnf clean all && \
    rm -rf /var/cache/dnf
