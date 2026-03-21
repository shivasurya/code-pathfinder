FROM docker:latest

# CRITICAL: Exposes Docker socket as volume
VOLUME ["/var/run/docker.sock"]

# This container can now control the host Docker daemon
CMD ["docker", "ps"]
