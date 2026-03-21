FROM centos:7

# Good: Using -y flag for non-interactive installation
RUN yum install -y httpd mod_ssl && \
    yum clean all && \
    rm -rf /var/cache/yum
