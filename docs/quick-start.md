# KrakenHashes Quick Start Guide

Get KrakenHashes up and running in under 5 minutes!

## Prerequisites

- Docker and Docker Compose installed
- 4GB RAM minimum
- Linux-based system recommended

## 1. Download configuration template

```bash
# Create a directory for KrakenHashes
mkdir krakenhashes && cd krakenhashes

# Download the environment template
wget https://raw.githubusercontent.com/yourusername/krakenhashes/main/.env.example
cp .env.example .env

# Edit the .env file and change these at minimum:
# - DB_PASSWORD (from default)
# - JWT_SECRET (from default)
# - PUID/PGID (to match your user: run 'id -u' and 'id -g')
```

## 2. Create docker-compose.yml

Create this `docker-compose.yml` file:

```yaml
services:
  postgres:
    image: postgres:15-alpine
    container_name: krakenhashes-postgres
    volumes:
      - postgres_data:/var/lib/postgresql/data
    environment:
      - POSTGRES_USER=${DB_USER}
      - POSTGRES_PASSWORD=${DB_PASSWORD}
      - POSTGRES_DB=${DB_NAME}
    restart: unless-stopped
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${DB_USER}"]
      interval: 5s
      retries: 5

  krakenhashes:
    image: zerkereod/krakenhashes:latest
    container_name: krakenhashes-app
    depends_on:
      postgres:
        condition: service_healthy
    env_file:
      - .env
    ports:
      - "${FRONTEND_PORT:-443}:443"
      - "${KH_HTTPS_PORT:-31337}:31337"
    volumes:
      - krakenhashes_data:/var/lib/krakenhashes
      - ./logs:/var/log/krakenhashes
    environment:
      - DB_HOST=postgres  # Override to use container name
      - PUID=${PUID}
      - PGID=${PGID}
    restart: unless-stopped

volumes:
  postgres_data:
  krakenhashes_data:
```

## 3. Start KrakenHashes

```bash
# Start the application
docker-compose up -d

# Wait for initialization (about 30 seconds)
docker-compose logs -f krakenhashes
```

## 4. Access the Application

Open your browser and navigate to:
- **https://localhost** (redirects to port 443)

**Note:** You'll see a certificate warning because we're using self-signed certificates. This is normal for local development.

## 5. First Login

1. Check the logs for the default admin credentials:
   ```bash
   docker-compose logs krakenhashes | grep -A5 "Admin user created"
   ```

2. Log in with the provided credentials
3. **Important:** Change the admin password immediately after first login

## 6. Quick Test

1. **Upload hashcat binary** (required first):
   - Navigate to Admin → Binaries
   - Click "Upload Binary"
   - Upload your hashcat binary (download from https://hashcat.net/hashcat/)
   - Select the appropriate platform (Linux, Windows, etc.)

2. **Upload a wordlist**:
   - Navigate to Admin → Wordlists
   - Click "Upload Wordlist"
   - Upload a small wordlist file (e.g., rockyou.txt or a test file)

3. **Create a test hashlist**:
   - Navigate to Hashlists
   - Click "Create Hashlist"
   - Add a few test hashes (e.g., MD5 hashes like `5f4dcc3b5aa765d61d8327deb882cf99` for "password")

4. **Create a job** (when agents are connected):
   - Navigate to Jobs
   - Select your hashlist
   - Choose a preset job template
   - Configure the attack settings
   - Start the job

**Note**: Jobs require at least one connected agent to execute. Without agents, jobs will remain in pending status.

## Common Tasks

### View Logs
```bash
# All logs
docker-compose logs -f

# Backend logs only
docker-compose logs -f krakenhashes | grep backend

# Check for errors
docker-compose logs | grep -i error
```

### Stop the Application
```bash
docker-compose down
```

### Update to Latest Version
```bash
# Pull latest image
docker pull zerkereod/krakenhashes:latest

# Restart with new image
docker-compose up -d
```

### Backup Database
```bash
docker exec krakenhashes-postgres pg_dump -U krakenhashes krakenhashes > backup.sql
```

## Troubleshooting

### Cannot access the web interface
1. Check if containers are running: `docker-compose ps`
2. Check logs for errors: `docker-compose logs`
3. Ensure ports 443 and 31337 are not in use: `netstat -tlnp | grep -E "443|31337"`

### Database connection errors
1. Ensure PostgreSQL is healthy: `docker-compose ps`
2. Check database logs: `docker-compose logs postgres`
3. Verify environment variables match in both services

### Certificate warnings
This is normal with self-signed certificates. For production, see the [Installation Guide](installation.md) for proper TLS setup.

## Next Steps

- **For Users**: Read [Understanding Jobs and Workflows](user/understanding_jobs_and_workflows.md)
- **For Admins**: Review the full [Installation Guide](installation.md) for production setup
- **For Developers**: See [Development Setup](installation.md#development-installation)

## Getting Help

- Check the [full documentation](README.md)
- Report issues on [GitHub](https://github.com/yourusername/krakenhashes/issues)
- Join our community chat (coming soon)