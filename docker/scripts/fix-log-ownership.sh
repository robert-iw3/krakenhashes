#!/bin/sh
# This script ensures log files are created with proper ownership

PUID=${PUID:-1000}
PGID=${PGID:-1000}

# Create log directories if they don't exist
mkdir -p /var/log/krakenhashes/backend
mkdir -p /var/log/krakenhashes/nginx

# Touch log files to ensure they exist
touch /var/log/krakenhashes/backend/backend.log
touch /var/log/krakenhashes/nginx/access.log
touch /var/log/krakenhashes/supervisord.log

# Set ownership based on PUID/PGID
chown ${PUID}:${PGID} /var/log/krakenhashes/backend/backend.log
chown www-data:www-data /var/log/krakenhashes/nginx/access.log
chown ${PUID}:${PGID} /var/log/krakenhashes/supervisord.log

# Set permissions
chmod 644 /var/log/krakenhashes/backend/backend.log
chmod 644 /var/log/krakenhashes/nginx/access.log
chmod 644 /var/log/krakenhashes/supervisord.log