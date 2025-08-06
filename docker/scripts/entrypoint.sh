#!/bin/sh
set -e

# Set default PUID/PGID if not provided
PUID=${PUID:-1000}
PGID=${PGID:-1000}

echo "Starting with UID: $PUID and GID: $PGID"

# Update krakenhashes user and group IDs
if [ "$PUID" != "1000" ] || [ "$PGID" != "1000" ]; then
    echo "Updating krakenhashes user/group to UID: $PUID, GID: $PGID"
    
    # Update group ID
    if [ "$PGID" != "1000" ]; then
        groupmod -g "$PGID" krakenhashes
    fi
    
    # Update user ID
    if [ "$PUID" != "1000" ]; then
        usermod -u "$PUID" krakenhashes
    fi
fi

# Fix ownership of directories that krakenhashes user needs to access
echo "Fixing ownership of directories..."
chown -R krakenhashes:krakenhashes /var/lib/krakenhashes || true
chown -R krakenhashes:krakenhashes /var/log/krakenhashes/backend || true
chown -R krakenhashes:krakenhashes /home/krakenhashes || true

# Fix ownership of config directory but keep certs readable
chown -R krakenhashes:krakenhashes /etc/krakenhashes || true
# Ensure the certificates directory exists and has proper permissions
mkdir -p /etc/krakenhashes/certs
chown krakenhashes:krakenhashes /etc/krakenhashes/certs
chmod 755 /etc/krakenhashes/certs

# If certificates already exist, ensure they are readable
if [ -n "$(ls -A /etc/krakenhashes/certs 2>/dev/null)" ]; then
    echo "Making existing certificates readable..."
    find /etc/krakenhashes/certs -type f -exec chmod 644 {} \;
    find /etc/krakenhashes/certs -type f -exec chown krakenhashes:krakenhashes {} \;
fi

# Create required log directories
for dir in backend nginx; do
    mkdir -p "/var/log/krakenhashes/$dir"
done

# Set permissions for log paths (but keep backend owned by krakenhashes)
chmod -R 755 "/var/log/krakenhashes"
chown -R www-data:www-data "/var/log/krakenhashes/nginx"
# Backend logs should remain owned by krakenhashes (already set above)

# Create backend .env file
cat > /etc/krakenhashes/.env << EOF
# Server Configuration
KH_HTTP_PORT=${KH_HTTP_PORT:-1337}
KH_HOST=${KH_HOST:-0.0.0.0}
KH_HTTPS_PORT=${KH_HTTPS_PORT:-31337}
KH_IN_DOCKER=TRUE

# Get container's hostname and IP
KH_ADDITIONAL_DNS_NAMES=localhost,$(hostname)
KH_ADDITIONAL_IP_ADDRESSES=127.0.0.1,0.0.0.0,$(hostname -i)

# Database Configuration
DB_HOST=${DB_HOST:-postgres}
DB_PORT=${DB_PORT:-5432}
DB_NAME=${DB_NAME}
DB_USER=${DB_USER}
DB_PASSWORD=${DB_PASSWORD}

# JWT Configuration
JWT_SECRET=${JWT_SECRET:-default_jwt_secret_replace_in_production}
JWT_EXPIRATION=${JWT_EXPIRATION:-24h}

# TLS Configuration
TLS_MODE=${TLS_MODE:-self-signed}
TLS_CERT_FILE=${TLS_CERT_FILE:-/etc/krakenhashes/certs/cert.pem}
TLS_KEY_FILE=${TLS_KEY_FILE:-/etc/krakenhashes/certs/key.pem}

# CORS Configuration
ALLOWED_ORIGINS=${ALLOWED_ORIGINS:-*}

# Logging Configuration
DEBUG=${DEBUG:-true}
LOG_LEVEL=${LOG_LEVEL:-DEBUG}

# Version Information
VERSION=${VERSION}
EOF

# Make sure the .env file is readable by krakenhashes
chown krakenhashes:krakenhashes /etc/krakenhashes/.env
chmod 644 /etc/krakenhashes/.env

# Wait for PostgreSQL to be ready (PostgreSQL runs in a separate container)
echo "Waiting for PostgreSQL at ${DB_HOST}:${DB_PORT}..."
until nc -z ${DB_HOST} ${DB_PORT} 2>/dev/null; do
    echo "PostgreSQL is unavailable - sleeping"
    sleep 2
done
echo "PostgreSQL is up - continuing with startup"

# Print environment variables for debugging
echo "Environment variables:"
echo "KH_IN_DOCKER=${KH_IN_DOCKER}"
echo "KH_HOST=${KH_HOST}"
echo "KH_HTTP_PORT=${KH_HTTP_PORT}"
echo "KH_HTTPS_PORT=${KH_HTTPS_PORT}"


echo "Starting supervisord..."
exec "$@" 