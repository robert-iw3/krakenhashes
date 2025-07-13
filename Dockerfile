# Build stage for frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend
# Install jq for version extraction
RUN apk add --no-cache jq
# Copy versions.json for build
COPY versions.json ./
# Install dependencies
COPY frontend/package*.json ./
RUN npm install --save-dev @babel/plugin-proposal-private-property-in-object && \
    npm ci
COPY frontend/ ./
RUN VERSION=$(jq -r .frontend versions.json) && \
    echo "REACT_APP_VERSION=$VERSION" >> .env && \
    CI=false npm run build

# Build stage for backend
FROM golang:1.23.1-alpine AS backend-builder
WORKDIR /app/backend
# Install jq for version extraction
RUN apk add --no-cache jq
# Copy versions.json for build
COPY versions.json ./
COPY backend/go.* ./
RUN go mod download
COPY backend/ ./
RUN VERSION=$(jq -r .backend versions.json) && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-X main.Version=$VERSION" -o krakenhashes ./cmd/server

# Final stage
FROM postgres:15-alpine

# Install required packages
RUN apk add --no-cache \
    postgresql15 \
    nginx \
    supervisor \
    certbot \
    logrotate \
    tzdata \
    jq \
    musl-locales \
    musl-locales-lang \
    openssl \
    p7zip \
    shadow \
    libcap \
    su-exec

# Create krakenhashes user and group with default UID/GID 1000
RUN addgroup -g 1000 -S krakenhashes && \
    adduser -u 1000 -S -G krakenhashes -h /home/krakenhashes -s /bin/sh krakenhashes

# Create necessary directories and set permissions
RUN set -ex && \
    # Add nginx and postgres users/groups if they don't exist
    addgroup -S nginx 2>/dev/null || true && \
    adduser -S -D -H -h /var/cache/nginx -s /sbin/nologin -G nginx -g nginx nginx 2>/dev/null || true && \
    # Create base directories
    mkdir -p /var/log/krakenhashes && \
    mkdir -p /etc/krakenhashes && \
    mkdir -p /usr/share/nginx/html && \
    mkdir -p /var/cache/nginx && \
    mkdir -p /var/run && \
    # Create log directories with proper ownership
    install -d -m 755 -o krakenhashes -g krakenhashes /var/log/krakenhashes/backend && \
    install -d -m 755 -o root -g root /var/log/krakenhashes/frontend && \
    install -d -m 755 -o nginx -g nginx /var/log/krakenhashes/nginx && \
    install -d -m 755 -o postgres -g postgres /var/log/krakenhashes/postgres && \
    # Create log files with proper ownership
    install -m 644 -o postgres -g postgres /dev/null /var/log/krakenhashes/postgres/stdout.log && \
    install -m 644 -o postgres -g postgres /dev/null /var/log/krakenhashes/postgres/stderr.log && \
    install -m 644 -o krakenhashes -g krakenhashes /dev/null /var/log/krakenhashes/backend/backend.log && \
    install -m 644 -o nginx -g nginx /dev/null /var/log/krakenhashes/nginx/access.log && \
    install -m 644 -o root -g root /dev/null /var/log/krakenhashes/logrotate.log && \
    install -m 644 -o root -g root /dev/null /var/log/krakenhashes/logrotate.err && \
    install -m 644 -o root -g root /dev/null /var/log/krakenhashes/supervisord.log

# Copy and protect versions.json in a non-persistent location
COPY versions.json /usr/local/share/krakenhashes/versions.json
RUN chown root:root /usr/local/share/krakenhashes/versions.json && \
    chmod 644 /usr/local/share/krakenhashes/versions.json

# Extract version for labels
ARG VERSION
RUN VERSION=$(jq -r .backend /usr/local/share/krakenhashes/versions.json)

# Add version labels
LABEL org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.description="KrakenHashes - Password Cracking Management System" \
      org.opencontainers.image.source="https://github.com/ZerkerEOD/krakenhashes"

# Copy PostgreSQL configuration
COPY docker/postgres/postgresql.conf /etc/postgresql/postgresql.conf

# Copy Nginx configuration
COPY docker/nginx/nginx.conf /etc/nginx/nginx.conf
COPY docker/nginx/default.conf /etc/nginx/conf.d/default.conf

# Copy logrotate configuration
COPY docker/logrotate/krakenhashes /etc/logrotate.d/krakenhashes

# Copy supervisord configuration
COPY docker/supervisord/supervisord.conf /etc/supervisord.conf

# Copy built artifacts
COPY --from=frontend-builder /app/frontend/build /usr/share/nginx/html
COPY --from=backend-builder /app/backend/krakenhashes /usr/local/bin/

# Copy migrations
COPY backend/db/migrations /usr/local/share/krakenhashes/migrations

# Create data directory with proper ownership
RUN mkdir -p /var/lib/krakenhashes && \
    chown -R krakenhashes:krakenhashes /var/lib/krakenhashes && \
    chmod 750 /var/lib/krakenhashes

# Give nginx capability to bind to port 443
RUN setcap 'cap_net_bind_service=+ep' /usr/sbin/nginx

# Copy startup scripts
COPY docker/scripts/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Set environment variables
ENV PGDATA=/var/lib/postgresql/data \
    POSTGRES_HOST_AUTH_METHOD=trust \
    KH_IN_DOCKER=TRUE \
    KH_HOST=0.0.0.0

# Expose ports
EXPOSE 443 1337 31337 5432

# Set entrypoint
ENTRYPOINT ["/entrypoint.sh"]
CMD ["supervisord", "-c", "/etc/supervisord.conf"] 