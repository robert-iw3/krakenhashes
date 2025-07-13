#!/bin/sh
# Deploy hook for certbot - called after successful renewal

set -e

echo "Certificates renewed, reloading services..."

# Reload nginx without downtime
nginx -s reload

# Signal backend to reload certificates
# The backend should handle SIGHUP to reload certificates
pkill -HUP krakenhashes || true

echo "Services reloaded successfully"