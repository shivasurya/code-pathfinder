FROM ubuntu:latest

# Missing USER instruction - should trigger DOCKER-SEC-001
# Using :latest tag - should trigger DOCKER-BP-001

RUN apt-get update && apt-get upgrade -y && \
    apt-get install -y nginx sudo

# Using sudo - should trigger DOCKER-SEC-007
RUN sudo chown -R www-data:www-data /var/www

# No pipefail - should trigger DOCKER-BP-010
RUN wget https://example.com/file.tar.gz | tar xz

# Missing yum clean all
RUN yum install -y httpd

# Last USER is root - should trigger DOCKER-SEC-009
USER root

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
