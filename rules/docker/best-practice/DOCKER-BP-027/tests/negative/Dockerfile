# Good: Let Docker handle platform selection automatically
FROM ubuntu:22.04
RUN apt-get update && apt-get install -y nginx

# Docker automatically pulls correct image for:
# - linux/amd64 on x86_64 systems
# - linux/arm64 on ARM systems
