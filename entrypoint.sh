#!/bin/bash
set -e

# Default settings
export PORT="${PORT:-8080}"
export UUID="${UUID:-$(cat /proc/sys/kernel/random/uuid 2>/dev/null || echo "23ad62a3-2c18-4286-a721-6898867ec43d")}"
export WSPATH="${WSPATH:-/vless}"

echo "================================================================"
echo "⚡ Starting Railway Xray VLESS VPN Server..."
echo "🌐 Port: ${PORT}"
echo "🔑 UUID: ${UUID}"
echo "🛣️ WebSocket Path: ${WSPATH}"
echo "================================================================"

# Replace variables in templates
sed -e "s|\$UUID|${UUID}|g" \
    -e "s|\$WSPATH|${WSPATH}|g" \
    /etc/xray/config.json.template > /etc/xray/config.json

sed -e "s|\$PORT|${PORT}|g" \
    -e "s|\$WSPATH|${WSPATH}|g" \
    /etc/nginx/nginx.conf.template > /etc/nginx/nginx.conf

sed -e "s|__UUID__|${UUID}|g" \
    -e "s|__WSPATH__|${WSPATH}|g" \
    /var/www/html/index.html.template > /var/www/html/index.html

# Start Xray core in the background
echo "🚀 Launching Xray Core..."
/usr/local/bin/xray run -c /etc/xray/config.json &

# Log connection info
DOMAIN="${RAILWAY_PUBLIC_DOMAIN:-${RAILWAY_STATIC_URL:-your-app.up.railway.app}}"
echo "================================================================"
echo "✅ Xray Core is running on port 10000!"
echo "📱 Your VLESS Link Template:"
echo "vless://${UUID}@${DOMAIN}:443?path=${WSPATH}&security=tls&encryption=none&type=ws#RailwayVPN"
echo "================================================================"

# Start Nginx in foreground
echo "🌐 Launching Nginx Reverse Proxy on port ${PORT}..."
exec nginx -g 'daemon off;'
