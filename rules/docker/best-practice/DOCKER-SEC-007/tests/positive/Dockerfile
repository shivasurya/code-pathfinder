FROM ubuntu:latest

# Installing sudo (unnecessary and dangerous)
RUN apt-get update && apt-get install -y sudo

# Using sudo in RUN (redundant - already root)
RUN sudo apt-get install -y nginx

# Creating sudoers file (privilege escalation risk)
RUN echo "appuser ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

USER appuser
# Now attacker with access can easily become root
CMD ["sudo", "/bin/bash"]
