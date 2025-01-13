#!/bin/sh
set -e

# Create required log directories
for dir in postgres backend nginx; do
    mkdir -p "/var/log/krakenhashes/$dir"
done

# Create required config directories
mkdir -p "/etc/krakenhashes/certs"

# Set permissions for log paths
chown -R root:root "/var/log/krakenhashes"
chmod -R 755 "/var/log/krakenhashes"

# Set permissions for config paths
chown -R root:root "/etc/krakenhashes"
chmod -R 755 "/etc/krakenhashes"
chmod 700 "/etc/krakenhashes/certs"

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
DB_HOST=${DB_HOST:-localhost}
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

# Start PostgreSQL
echo "Starting PostgreSQL..."
if [ ! -s "$PGDATA/PG_VERSION" ]; then
    echo "Initializing PostgreSQL database..."
    install -d -m 700 -o postgres -g postgres "$PGDATA"
    su postgres -c "initdb -D $PGDATA -U postgres"
    
    # Configure PostgreSQL authentication
    echo "Configuring PostgreSQL authentication..."
    echo "host all all 0.0.0.0/0 md5" >> "$PGDATA/pg_hba.conf"
    echo "local all all md5" >> "$PGDATA/pg_hba.conf"
    
    # Start PostgreSQL temporarily to create user and database
    echo "Starting PostgreSQL temporarily to create user and database..."
    su postgres -c "pg_ctl -D $PGDATA -o '-c listen_addresses=*' start"
    
    # Wait for PostgreSQL to be ready
    until su postgres -c "pg_isready -h localhost"; do
        echo "Waiting for PostgreSQL to be ready..."
        sleep 1
    done
    
    echo "Creating database user and database..."
    su postgres -c "psql -v ON_ERROR_STOP=1" << EOF
        CREATE USER ${DB_USER} WITH PASSWORD '${DB_PASSWORD}';
        CREATE DATABASE ${DB_NAME} OWNER ${DB_USER};
        GRANT ALL PRIVILEGES ON DATABASE ${DB_NAME} TO ${DB_USER};
EOF
    
    # Stop PostgreSQL to apply configuration
    su postgres -c "pg_ctl -D $PGDATA stop"
fi

# Start PostgreSQL with full configuration
echo "Starting PostgreSQL..."
su postgres -c "pg_ctl -D $PGDATA -w start"

# Wait for PostgreSQL to be ready
until su postgres -c "pg_isready -h localhost -U ${DB_USER}"; do
    echo "Waiting for PostgreSQL to be ready..."
    sleep 1
done
echo "PostgreSQL is up - executing command"

# Print environment variables for debugging
echo "Environment variables:"
echo "KH_IN_DOCKER=${KH_IN_DOCKER}"
echo "KH_HOST=${KH_HOST}"
echo "KH_HTTP_PORT=${KH_HTTP_PORT}"
echo "KH_HTTPS_PORT=${KH_HTTPS_PORT}"

# Ensure postgres user can write to its log directory
chown -R postgres:postgres "/var/log/krakenhashes/postgres"

echo "Starting supervisord..."
exec "$@" 