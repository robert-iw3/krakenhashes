# KrakenHashes Installation Guide

This guide covers both production and development installation of KrakenHashes.

## Production Installation

### Prerequisites

- Docker Engine 20.10+ and Docker Compose 2.0+
- 4GB+ RAM recommended
- 20GB+ disk space for hash files, wordlists, and rules
- Linux host (Ubuntu 20.04+, Debian 11+, RHEL 8+, or similar)

### Why Use .env.example?

The `.env.example` file is the authoritative source for all configuration options. Benefits include:

- **Complete Configuration**: Contains all available settings with descriptions
- **Secure Defaults**: Includes proper UID/GID settings and certificate configurations
- **Version Compatibility**: Always matches the current version's requirements
- **Documentation**: Each variable is documented inline
- **Best Practices**: Pre-configured with recommended settings

### Quick Start with Docker Hub

The easiest way to run KrakenHashes is using the pre-built Docker image from Docker Hub.

#### 1. Download the environment template

```bash
# Download the .env.example file
wget https://raw.githubusercontent.com/yourusername/krakenhashes/main/.env.example

# Or using curl
curl -O https://raw.githubusercontent.com/yourusername/krakenhashes/main/.env.example

# Copy it to .env
cp .env.example .env
```

#### 2. Configure your environment

Edit the `.env` file and update these critical settings:

```bash
# IMPORTANT: Change these from defaults
DB_PASSWORD=your-secure-password-here
JWT_SECRET=your-very-long-random-string-here

# Set your user/group IDs (run 'id -u' and 'id -g' to find yours)
PUID=1000
PGID=1000

# For production, consider changing:
DEBUG=false
LOG_LEVEL=INFO
KH_TLS_MODE=self-signed  # or 'provided' if using your own certs
```

#### 3. Create a docker-compose.yml file

```yaml
services:
  postgres:
    image: postgres:15-alpine
    container_name: krakenhashes-postgres
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ${LOG_DIR:-./logs}/postgres:/var/log/postgresql
    environment:
      - POSTGRES_USER=${DB_USER:-krakenhashes}
      - POSTGRES_PASSWORD=${DB_PASSWORD:-krakenhashes}
      - POSTGRES_DB=${DB_NAME:-krakenhashes}
    restart: unless-stopped
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${DB_USER:-krakenhashes}"]
      interval: 5s
      timeout: 5s
      retries: 5

  krakenhashes:
    image: zerkereod/krakenhashes:latest
    container_name: krakenhashes-app
    depends_on:
      postgres:
        condition: service_healthy
    env_file:
      - .env  # Loads all variables from .env file
    ports:
      - "${FRONTEND_PORT:-443}:443"
      - "${KH_HTTPS_PORT:-31337}:31337"
    volumes:
      - krakenhashes_data:/var/lib/krakenhashes
      - ${LOG_DIR:-./logs}:/var/log/krakenhashes
      - ${KH_CONFIG_DIR:-./config}:/etc/krakenhashes
    environment:
      # Override specific values for container networking
      - DB_HOST=postgres  # Use container name instead of localhost
      - PUID=${PUID:-1000}
      - PGID=${PGID:-1000}
    restart: unless-stopped

volumes:
  postgres_data:
  krakenhashes_data:
```

**Note**: The `env_file` directive loads ALL variables from `.env` into the container. The `environment` section then overrides specific values as needed (e.g., `DB_HOST=postgres` for container networking).

#### Alternative: Without env_file

If you prefer not to load all variables, you can specify only what's needed:

```yaml
krakenhashes:
  image: zerkereod/krakenhashes:latest
  environment:
    - DB_HOST=postgres
    - DB_PORT=${DB_PORT:-5432}
    - DB_NAME=${DB_NAME:-krakenhashes}
    - DB_USER=${DB_USER:-krakenhashes}
    - DB_PASSWORD=${DB_PASSWORD}  # No default for security
    - JWT_SECRET=${JWT_SECRET}     # No default for security
    - PUID=${PUID:-1000}
    - PGID=${PGID:-1000}
    - DEBUG=${DEBUG:-false}
    - LOG_LEVEL=${LOG_LEVEL:-INFO}
    - KH_TLS_MODE=${KH_TLS_MODE:-self-signed}
    # Add other variables as needed
```

However, using `env_file` is recommended as it ensures all configuration options are available to the application.

#### 4. Start KrakenHashes

```bash
# Pull the latest image
docker pull zerkereod/krakenhashes:latest

# Start the services
docker-compose up -d

# Check logs
docker-compose logs -f
```

#### 5. Access the Application

- Frontend: https://localhost:443
- Backend API: https://localhost:31337
- Default admin credentials are created on first run (check logs)

#### 6. Initial Setup

Before you can run password cracking jobs, you need to:

1. **Upload Hashcat Binary**:
   - Download hashcat from https://hashcat.net/hashcat/
   - Navigate to Admin → Binaries
   - Upload the hashcat binary for your target platforms

2. **Upload Wordlists** (optional but recommended):
   - Navigate to Admin → Wordlists
   - Upload common wordlists (e.g., rockyou.txt)

3. **Configure Agents** (for job execution):
   - Agents connect to the backend to execute jobs
   - See [Agent Documentation](admin/agent_file_sync.md) for setup

### How Environment Variables Work

Docker Compose processes environment variables in this order:
1. **Shell environment** - Variables already set in your shell
2. **`.env` file** - Variables defined in the .env file (used for variable substitution in docker-compose.yml)
3. **`env_file`** - All variables from specified files are loaded into the container
4. **`environment`** - Explicit values that override previous sources

In our setup:
- `${VARIABLE:-default}` in docker-compose.yml uses values from .env or defaults
- `env_file: - .env` passes ALL variables from .env into the container
- `environment:` section overrides specific values (like DB_HOST for container networking)

### Managing Multiple Environments

You can maintain separate configurations for different environments:

#### Option 1: Multiple .env files

```bash
# Create environment-specific files
cp .env.example .env.prod
cp .env.example .env.dev

# Use them with docker-compose
docker-compose --env-file .env.prod up -d
docker-compose --env-file .env.dev up -d
```

#### Option 2: Environment prefixes

```bash
# Production
cp .env.example prod.env
docker-compose -p krakenhashes-prod --env-file prod.env up -d

# Development
cp .env.example dev.env
docker-compose -p krakenhashes-dev --env-file dev.env up -d
```

#### Key differences between environments:

**Production (.env.prod)**:
```bash
DEBUG=false
LOG_LEVEL=INFO
KH_TLS_MODE=provided  # Use real certificates
DB_PASSWORD=strong-production-password
JWT_SECRET=very-long-random-production-secret
```

**Development (.env.dev)**:
```bash
DEBUG=true
LOG_LEVEL=DEBUG
KH_TLS_MODE=self-signed
DEBUG_SQL=true
DEBUG_HTTP=true
```

### Production Configuration

#### TLS/SSL Options

KrakenHashes supports three TLS modes:

1. **self-signed** (default) - Automatically generates self-signed certificates
2. **provided** - Use your own certificates (not yet implemented)
3. **certbot** - Automatically obtain Let's Encrypt certificates via DNS-01 challenge

##### Using Let's Encrypt Certificates (Certbot)

For production deployments with valid certificates, see the [Certbot Setup Guide](certbot-setup.md).

```bash
# Quick setup for certbot mode
KH_TLS_MODE=certbot
KH_CERTBOT_DOMAIN=your-domain.com
KH_CERTBOT_EMAIL=admin@your-domain.com
CLOUDFLARE_API_TOKEN=your-cloudflare-token
```


#### Data Persistence

Important directories that should be persisted:

- `/var/lib/krakenhashes` - Application data (binaries, wordlists, rules, hashlists)
- `/var/log/krakenhashes` - Application logs
- PostgreSQL data volume

#### Environment Variables

The `.env.example` file contains all available configuration options with descriptions. Key variables include:

**Essential Security Settings**:
| Variable | Default | Description |
|----------|---------|-------------|
| `DB_PASSWORD` | krakenhashes | **MUST CHANGE** - Database password |
| `JWT_SECRET` | change_this_in_production | **MUST CHANGE** - JWT signing secret |
| `PUID` | 1000 | User ID for file permissions (run `id -u`) |
| `PGID` | 1000 | Group ID for file permissions (run `id -g`) |

**Database Configuration**:
| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | localhost | PostgreSQL hostname (use 'postgres' for Docker) |
| `DB_PORT` | 5432 | PostgreSQL port |
| `DB_NAME` | krakenhashes | Database name |
| `DB_USER` | krakenhashes | Database user |

**TLS/Certificate Settings**:
| Variable | Default | Description |
|----------|---------|-------------|
| `KH_TLS_MODE` | self-signed | TLS mode: self-signed, provided, certbot |
| `KH_CA_ORGANIZATION` | KrakenHashes | Organization name for certificates |
| `KH_CERT_VALIDITY_DAYS` | 365 | Certificate validity period |
| `KH_ADDITIONAL_DNS_NAMES` | localhost,krakenhashes.local | Additional DNS names for certificate |

**Debug Settings** (Development):
| Variable | Default | Description |
|----------|---------|-------------|
| `DEBUG` | false | Enable debug mode |
| `DEBUG_SQL` | false | Log SQL queries |
| `DEBUG_HTTP` | false | Log HTTP requests |
| `LOG_LEVEL` | INFO | Logging level: DEBUG, INFO, WARNING, ERROR |

See `.env.example` for the complete list of configuration options.

### Production Best Practices

1. **Security**
   - Always change default passwords
   - Use a strong JWT_SECRET (minimum 32 characters)
   - Use proper TLS certificates for production
   - Restrict network access to necessary ports only

2. **Initial Data Setup**
   - Upload hashcat binaries for all platforms your agents will use
   - Pre-load common wordlists (rockyou.txt, SecLists, etc.)
   - Configure standard rule files
   - Create preset job templates for common attack patterns

3. **Backup**
   - Regular PostgreSQL backups: `docker exec krakenhashes-postgres pg_dump -U krakenhashes krakenhashes > backup.sql`
   - Backup the data volume: `/var/lib/krakenhashes`

4. **Monitoring**
   - Monitor logs in `/var/log/krakenhashes`
   - Set up health checks for the application endpoints
   - Monitor disk space for hash storage

5. **Updates**
   ```bash
   # Pull latest image
   docker pull zerkereod/krakenhashes:latest
   
   # Recreate container with new image
   docker-compose up -d --force-recreate krakenhashes
   ```

### Troubleshooting Production Issues

#### Container won't start
```bash
# Check logs
docker-compose logs krakenhashes

# Check if ports are already in use
netstat -tlnp | grep -E "443|31337|1337|5432"
```

#### Database connection issues
```bash
# Test database connectivity
docker exec krakenhashes-app nc -zv postgres 5432

# Check database logs
docker-compose logs postgres
```

#### Permission issues
```bash
# Fix ownership (adjust PUID/PGID as needed)
docker exec krakenhashes-app chown -R 1000:1000 /var/lib/krakenhashes
```

---

## Development Installation

### Prerequisites

- Docker Engine 20.10+ and Docker Compose 2.0+
- Go 1.23.1+
- Node.js 20+
- Git
- 8GB+ RAM recommended for development

### Development Setup Options

#### Option 1: Docker Development Environment (Recommended)

This setup provides hot-reloading for both backend and frontend.

1. **Clone the repository**
   ```bash
   git clone https://github.com/yourusername/krakenhashes.git
   cd krakenhashes
   ```

2. **Start development environment**
   ```bash
   # Start all services with hot-reloading
   docker-compose -f docker-compose.dev.yml up
   
   # Or run in background
   docker-compose -f docker-compose.dev.yml up -d
   ```

3. **Access the services**
   - Frontend: http://localhost:3000 (with hot-reload)
   - Backend API: https://localhost:31337
   - PostgreSQL: localhost:5432

4. **View logs**
   ```bash
   # All services
   docker-compose -f docker-compose.dev.yml logs -f
   
   # Specific service
   docker-compose -f docker-compose.dev.yml logs -f backend
   ```

The development environment features:
- **Backend**: Uses Air for Go hot-reloading
- **Frontend**: Uses React development server with hot-reload
- **Database**: PostgreSQL with persistent volume
- **Volumes**: Source code mounted for live updates

#### Option 2: Local Development (Traditional)

1. **Clone the repository**
   ```bash
   git clone https://github.com/yourusername/krakenhashes.git
   cd krakenhashes
   ```

2. **Start PostgreSQL**
   ```bash
   cd backend
   docker-compose up -d
   cd ..
   ```

3. **Run the backend**
   ```bash
   cd backend
   go mod download
   go run cmd/server/main.go
   ```

4. **Run the frontend**
   ```bash
   cd frontend
   npm install
   npm start
   ```

### Development Configuration

#### Backend Development

The backend uses Air for hot-reloading. Configuration is in `backend/.air.toml`:

```toml
[build]
  cmd = "go build -o ./tmp/main ./cmd/server"
  bin = "./tmp/main"
  include_ext = ["go", "tpl", "tmpl", "html"]
```

Environment variables for development:
```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_NAME=krakenhashes
export DB_USER=krakenhashes
export DB_PASSWORD=krakenhashes
export JWT_SECRET=dev_jwt_secret
export DEBUG=true
export LOG_LEVEL=DEBUG
```

#### Frontend Development

The frontend uses Create React App with environment variables:
```bash
REACT_APP_API_URL=https://localhost:31337
REACT_APP_WS_URL=wss://localhost:31337
REACT_APP_DEBUG=true
```

### Development Workflows

#### Running Tests

```bash
# Backend tests
cd backend
go test ./...
go test -v ./internal/services

# Frontend tests
cd frontend
npm test
```

#### Building for Production

```bash
# Build production Docker image
docker build -f Dockerfile.prod -t krakenhashes:local .

# Test production build locally
docker-compose down
docker-compose up -d
```

#### Database Migrations

```bash
# Apply migrations
cd backend
make migrate-up

# Rollback migrations
make migrate-down

# Create new migration
make migrate-create name=add_new_table
```

### Development Tools

#### Logging

View development logs with filtering:
```bash
# Backend logs
tail -f logs/backend.log | grep -i error

# Frontend logs
tail -f logs/frontend.log

# All logs
grep -i error logs/*.log
```

#### Database Access

```bash
# Connect to development database
docker exec -it krakenhashes-postgres-dev psql -U krakenhashes -d krakenhashes

# Quick query
docker exec krakenhashes-postgres-dev psql -U krakenhashes -d krakenhashes -c "SELECT * FROM users;"
```

### Switching Between Development and Production

```bash
# Stop development environment
docker-compose -f docker-compose.dev.yml down

# Start production environment
docker-compose up -d

# Switch back to development
docker-compose down
docker-compose -f docker-compose.dev.yml up
```

### Common Development Issues

#### Port conflicts
```bash
# Check what's using the ports
lsof -i :3000   # Frontend
lsof -i :31337  # Backend HTTPS
lsof -i :5432   # PostgreSQL
```

#### Go module issues
```bash
# Clear module cache
go clean -modcache

# Update dependencies
go mod tidy
go mod download
```

#### Frontend dependency issues
```bash
# Clear npm cache
cd frontend
rm -rf node_modules package-lock.json
npm cache clean --force
npm install
```

## Next Steps

- Review the [Configuration Guide](configuration.md) for detailed settings
- Check the [Admin Documentation](admin/) for system administration
- See [User Documentation](user/) for using KrakenHashes
- Join our community for support and updates