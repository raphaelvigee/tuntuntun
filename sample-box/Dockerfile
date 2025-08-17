# Use a standard Ubuntu base image
FROM ubuntu:latest

# Avoid interactive prompts during package installation
ENV DEBIAN_FRONTEND=noninteractive

# Install Nginx, OpenSSH server, and Supervisor
RUN apt-get update && apt-get install -y \
    nginx \
    openssh-server \
    supervisor \
    iperf3 \
    && rm -rf /var/lib/apt/lists/*

# --- SSH Configuration ---
RUN useradd -m -s /bin/bash test && echo "test:test" | chpasswd
RUN mkdir /var/run/sshd
RUN sed -i 's/#PasswordAuthentication yes/PasswordAuthentication yes/' /etc/ssh/sshd_config && \
    sed -i 's/#PermitRootLogin prohibit-password/PermitRootLogin yes/' /etc/ssh/sshd_config

# --- Nginx Configuration ---
RUN rm /etc/nginx/sites-enabled/default
COPY nginx.conf /etc/nginx/sites-available/
RUN ln -s /etc/nginx/sites-available/nginx.conf /etc/nginx/sites-enabled/

# --- Main Process ---
COPY tuntuntun /usr/local/bin/tuntuntun
RUN chmod +x /usr/local/bin/tuntuntun

# --- Supervisor Configuration ---
# Copy the supervisor config file
COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf

EXPOSE 5201

# Run supervisord as the main command
CMD ["/usr/bin/supervisord"]
