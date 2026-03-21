FROM ubuntu:22.04

# No USER instruction - container runs as root
RUN apt-get update && apt-get install -y nginx
COPY app /app
CMD ["nginx", "-g", "daemon off;"]
