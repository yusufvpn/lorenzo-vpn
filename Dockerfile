FROM alpine:latest

# Install dependencies: nginx, curl, unzip, bash
RUN apk add --no-cache nginx curl unzip bash

# Create directories
RUN mkdir -p /etc/xray /usr/local/bin /var/www/html /run/nginx /var/log/nginx

# Download and install Xray Core
RUN XRAY_URL="https://github.com/XTLS/Xray-core/releases/latest/download/Xray-linux-64.zip" && \
    curl -sSL -H "User-Agent: Mozilla/5.0" -o /tmp/xray.zip ${XRAY_URL} && \
    unzip -q /tmp/xray.zip -d /tmp/xray && \
    mv /tmp/xray/xray /usr/local/bin/xray && \
    chmod +x /usr/local/bin/xray && \
    rm -rf /tmp/xray /tmp/xray.zip

# Copy configuration templates and scripts
COPY config.json.template /etc/xray/config.json.template
COPY nginx.conf.template /etc/nginx/nginx.conf.template
COPY entrypoint.sh /entrypoint.sh
COPY web/ /var/www/html/

# Permissions
RUN chmod +x /entrypoint.sh

# Default environment variables
ENV PORT=8080
ENV UUID=""
ENV WSPATH="/vless"

EXPOSE 8080

ENTRYPOINT ["/entrypoint.sh"]
