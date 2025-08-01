# Backup and Restore Procedures

This guide provides comprehensive backup and restore procedures for KrakenHashes, covering database backups, file system backups, Docker volumes, and automated backup strategies.

## Overview

KrakenHashes stores critical data in multiple locations that require regular backups:

1. **PostgreSQL Database** - Contains all system metadata, user accounts, job configurations, and hash crack results
2. **File Storage** - Binary files, wordlists, rules, and uploaded hashlists
3. **Docker Volumes** - Persistent storage for containerized deployments
4. **Configuration Files** - TLS certificates and system configuration

## Data Locations

### Database

- **Container**: `krakenhashes-postgres`
- **Volume**: `krakenhashes_postgres_data`
- **Critical Tables**:
  - `users` - User accounts and authentication
  - `agents` - Registered compute agents
  - `hashlists` - Hash collections metadata
  - `hashes` - Individual hash records and crack status
  - `clients` - Customer/engagement tracking
  - `job_executions` - Job execution history
  - `wordlists`, `rules` - Attack resource metadata
  - `auth_tokens` - Authentication tokens
  - `vouchers` - Agent registration codes

### File Storage

Default location: `/var/lib/krakenhashes/` (configurable via `KH_DATA_DIR`)

```
/var/lib/krakenhashes/
├── binaries/          # Hashcat and other tool binaries
├── wordlists/         # Wordlist files
│   ├── general/
│   ├── specialized/
│   ├── targeted/
│   └── custom/
├── rules/             # Rule files
│   ├── hashcat/
│   ├── john/
│   └── custom/
├── hashlists/         # Uploaded hash files
└── hashlist_uploads/  # Temporary upload directory
```

### Docker Volumes

- `krakenhashes_postgres_data` - PostgreSQL data
- `krakenhashes_app_data` - Application data files

## Backup Strategies

### 1. PostgreSQL Database Backup

#### Manual Backup

```bash
# Create backup directory
mkdir -p /backup/krakenhashes/postgres/$(date +%Y%m%d)

# Backup using pg_dump (recommended for portability)
docker exec krakenhashes-postgres pg_dump \
  -U krakenhashes \
  -d krakenhashes \
  --verbose \
  --format=custom \
  --file=/tmp/krakenhashes_$(date +%Y%m%d_%H%M%S).dump

# Copy backup from container
docker cp krakenhashes-postgres:/tmp/krakenhashes_$(date +%Y%m%d_%H%M%S).dump \
  /backup/krakenhashes/postgres/$(date +%Y%m%d)/

# Alternative: Direct backup with compression
docker exec krakenhashes-postgres pg_dump \
  -U krakenhashes \
  -d krakenhashes \
  | gzip > /backup/krakenhashes/postgres/$(date +%Y%m%d)/krakenhashes_$(date +%Y%m%d_%H%M%S).sql.gz
```

#### Automated Database Backup Script

Create `/usr/local/bin/krakenhashes-db-backup.sh`:

```bash
#!/bin/bash

# KrakenHashes Database Backup Script
# Run via cron for automated backups

set -e

# Configuration
BACKUP_DIR="/backup/krakenhashes/postgres"
RETENTION_DAYS=30
DB_CONTAINER="krakenhashes-postgres"
DB_NAME="krakenhashes"
DB_USER="krakenhashes"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
DATE_DIR=$(date +%Y%m%d)

# Create backup directory
mkdir -p "${BACKUP_DIR}/${DATE_DIR}"

# Function to log messages
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

# Check if container is running
if ! docker ps | grep -q "${DB_CONTAINER}"; then
    log "ERROR: Container ${DB_CONTAINER} is not running"
    exit 1
fi

log "Starting database backup..."

# Perform backup
BACKUP_FILE="${BACKUP_DIR}/${DATE_DIR}/krakenhashes_${TIMESTAMP}.dump"
if docker exec "${DB_CONTAINER}" pg_dump \
    -U "${DB_USER}" \
    -d "${DB_NAME}" \
    --verbose \
    --format=custom \
    --file="/tmp/backup_${TIMESTAMP}.dump"; then
    
    # Copy backup from container
    docker cp "${DB_CONTAINER}:/tmp/backup_${TIMESTAMP}.dump" "${BACKUP_FILE}"
    
    # Clean up temp file in container
    docker exec "${DB_CONTAINER}" rm "/tmp/backup_${TIMESTAMP}.dump"
    
    # Compress backup
    gzip "${BACKUP_FILE}"
    
    log "Database backup completed: ${BACKUP_FILE}.gz"
    
    # Calculate backup size
    SIZE=$(du -h "${BACKUP_FILE}.gz" | cut -f1)
    log "Backup size: ${SIZE}"
else
    log "ERROR: Database backup failed"
    exit 1
fi

# Clean up old backups
log "Cleaning up backups older than ${RETENTION_DAYS} days..."
find "${BACKUP_DIR}" -type f -name "*.dump.gz" -mtime +${RETENTION_DAYS} -delete

log "Backup process completed"
```

Make it executable:
```bash
chmod +x /usr/local/bin/krakenhashes-db-backup.sh
```

### 2. File System Backup

#### Manual Backup

```bash
# Create backup directory
mkdir -p /backup/krakenhashes/files/$(date +%Y%m%d)

# Backup data directory with compression
tar -czf /backup/krakenhashes/files/$(date +%Y%m%d)/krakenhashes_data_$(date +%Y%m%d_%H%M%S).tar.gz \
  -C /var/lib/krakenhashes \
  binaries wordlists rules hashlists hashlist_uploads

# Backup with progress indicator
tar -czf /backup/krakenhashes/files/$(date +%Y%m%d)/krakenhashes_data_$(date +%Y%m%d_%H%M%S).tar.gz \
  --checkpoint=1000 \
  --checkpoint-action=echo="Processed %{r}T files" \
  -C /var/lib/krakenhashes \
  binaries wordlists rules hashlists hashlist_uploads
```

#### Automated File Backup Script

Create `/usr/local/bin/krakenhashes-file-backup.sh`:

```bash
#!/bin/bash

# KrakenHashes File System Backup Script
# Backs up all data files including binaries, wordlists, rules, and hashlists

set -e

# Configuration
BACKUP_DIR="/backup/krakenhashes/files"
DATA_DIR="${KH_DATA_DIR:-/var/lib/krakenhashes}"
RETENTION_DAYS=30
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
DATE_DIR=$(date +%Y%m%d)

# Create backup directory
mkdir -p "${BACKUP_DIR}/${DATE_DIR}"

# Function to log messages
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

# Check if data directory exists
if [ ! -d "${DATA_DIR}" ]; then
    log "ERROR: Data directory ${DATA_DIR} does not exist"
    exit 1
fi

log "Starting file system backup..."
log "Source directory: ${DATA_DIR}"

# Calculate total size
TOTAL_SIZE=$(du -sh "${DATA_DIR}" | cut -f1)
log "Total data size: ${TOTAL_SIZE}"

# Create backup
BACKUP_FILE="${BACKUP_DIR}/${DATE_DIR}/krakenhashes_data_${TIMESTAMP}.tar.gz"

# Backup with progress
tar -czf "${BACKUP_FILE}" \
    --checkpoint=1000 \
    --checkpoint-action=echo="[$(date '+%Y-%m-%d %H:%M:%S')] Processed %{r}T files" \
    -C "${DATA_DIR}" \
    binaries wordlists rules hashlists hashlist_uploads 2>&1 | while read line; do
        log "$line"
    done

if [ ${PIPESTATUS[0]} -eq 0 ]; then
    log "File backup completed: ${BACKUP_FILE}"
    
    # Calculate backup size
    SIZE=$(du -h "${BACKUP_FILE}" | cut -f1)
    log "Backup size: ${SIZE}"
    
    # Create checksum
    sha256sum "${BACKUP_FILE}" > "${BACKUP_FILE}.sha256"
    log "Checksum created: ${BACKUP_FILE}.sha256"
else
    log "ERROR: File backup failed"
    exit 1
fi

# Clean up old backups
log "Cleaning up backups older than ${RETENTION_DAYS} days..."
find "${BACKUP_DIR}" -type f \( -name "*.tar.gz" -o -name "*.sha256" \) -mtime +${RETENTION_DAYS} -delete

log "File backup process completed"
```

Make it executable:
```bash
chmod +x /usr/local/bin/krakenhashes-file-backup.sh
```

### 3. Docker Volume Backup

#### Manual Volume Backup

```bash
# Stop services to ensure consistency
docker-compose down

# Backup PostgreSQL volume
docker run --rm \
  -v krakenhashes_postgres_data:/source:ro \
  -v /backup/krakenhashes/volumes:/backup \
  alpine tar -czf /backup/postgres_data_$(date +%Y%m%d_%H%M%S).tar.gz -C /source .

# Backup application data volume
docker run --rm \
  -v krakenhashes_app_data:/source:ro \
  -v /backup/krakenhashes/volumes:/backup \
  alpine tar -czf /backup/app_data_$(date +%Y%m%d_%H%M%S).tar.gz -C /source .

# Restart services
docker-compose up -d
```

#### Automated Volume Backup Script

Create `/usr/local/bin/krakenhashes-volume-backup.sh`:

```bash
#!/bin/bash

# KrakenHashes Docker Volume Backup Script
# Backs up Docker volumes with minimal downtime

set -e

# Configuration
BACKUP_DIR="/backup/krakenhashes/volumes"
RETENTION_DAYS=30
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
DATE_DIR=$(date +%Y%m%d)
COMPOSE_FILE="/path/to/krakenhashes/docker-compose.yml"

# Create backup directory
mkdir -p "${BACKUP_DIR}/${DATE_DIR}"

# Function to log messages
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

# Function to backup a volume
backup_volume() {
    local volume_name=$1
    local backup_name=$2
    
    log "Backing up volume: ${volume_name}"
    
    docker run --rm \
        -v "${volume_name}:/source:ro" \
        -v "${BACKUP_DIR}/${DATE_DIR}:/backup" \
        alpine tar -czf "/backup/${backup_name}_${TIMESTAMP}.tar.gz" -C /source .
    
    if [ $? -eq 0 ]; then
        log "Volume backup completed: ${backup_name}_${TIMESTAMP}.tar.gz"
        
        # Create checksum
        cd "${BACKUP_DIR}/${DATE_DIR}"
        sha256sum "${backup_name}_${TIMESTAMP}.tar.gz" > "${backup_name}_${TIMESTAMP}.tar.gz.sha256"
    else
        log "ERROR: Failed to backup volume ${volume_name}"
        return 1
    fi
}

log "Starting Docker volume backup..."

# For live backups without downtime (PostgreSQL only)
if docker ps | grep -q "krakenhashes-postgres"; then
    log "Creating PostgreSQL checkpoint for consistent backup..."
    docker exec krakenhashes-postgres psql -U krakenhashes -c "CHECKPOINT;"
fi

# Backup volumes
backup_volume "krakenhashes_postgres_data" "postgres_data"
backup_volume "krakenhashes_app_data" "app_data"

# Clean up old backups
log "Cleaning up backups older than ${RETENTION_DAYS} days..."
find "${BACKUP_DIR}" -type f \( -name "*.tar.gz" -o -name "*.sha256" \) -mtime +${RETENTION_DAYS} -delete

log "Volume backup process completed"
```

### 4. Configuration Backup

```bash
#!/bin/bash

# Backup configuration files
BACKUP_DIR="/backup/krakenhashes/config/$(date +%Y%m%d)"
mkdir -p "${BACKUP_DIR}"

# Backup TLS certificates
tar -czf "${BACKUP_DIR}/certs_$(date +%Y%m%d_%H%M%S).tar.gz" \
  -C /etc/krakenhashes certs

# Backup environment files
cp /path/to/krakenhashes/.env "${BACKUP_DIR}/env_$(date +%Y%m%d_%H%M%S)"

# Backup docker-compose configuration
cp /path/to/krakenhashes/docker-compose.yml "${BACKUP_DIR}/"
```

## Automated Backup Schedule

Add to crontab (`crontab -e`):

```cron
# KrakenHashes Automated Backups
# Database backup - every 6 hours
0 */6 * * * /usr/local/bin/krakenhashes-db-backup.sh >> /var/log/krakenhashes-backup.log 2>&1

# File system backup - daily at 2 AM
0 2 * * * /usr/local/bin/krakenhashes-file-backup.sh >> /var/log/krakenhashes-backup.log 2>&1

# Volume backup - daily at 3 AM
0 3 * * * /usr/local/bin/krakenhashes-volume-backup.sh >> /var/log/krakenhashes-backup.log 2>&1

# Configuration backup - weekly on Sunday at 4 AM
0 4 * * 0 /usr/local/bin/krakenhashes-config-backup.sh >> /var/log/krakenhashes-backup.log 2>&1
```

## Restore Procedures

### 1. Database Restore

#### From pg_dump backup

```bash
# Stop the application
docker-compose stop krakenhashes

# Restore database
docker exec -i krakenhashes-postgres pg_restore \
  -U krakenhashes \
  -d krakenhashes \
  --clean \
  --if-exists \
  --verbose \
  < /backup/krakenhashes/postgres/20240101/krakenhashes_20240101_120000.dump

# Or from compressed SQL
gunzip -c /backup/krakenhashes/postgres/20240101/krakenhashes_20240101_120000.sql.gz | \
  docker exec -i krakenhashes-postgres psql -U krakenhashes -d krakenhashes

# Restart application
docker-compose start krakenhashes
```

#### Full database recreation

```bash
# Stop all services
docker-compose down

# Remove old database volume
docker volume rm krakenhashes_postgres_data

# Recreate and start database
docker-compose up -d postgres

# Wait for database to be ready
sleep 10

# Create database and user
docker exec krakenhashes-postgres psql -U postgres -c "CREATE DATABASE krakenhashes;"
docker exec krakenhashes-postgres psql -U postgres -c "CREATE USER krakenhashes WITH PASSWORD 'your-password';"
docker exec krakenhashes-postgres psql -U postgres -c "GRANT ALL PRIVILEGES ON DATABASE krakenhashes TO krakenhashes;"

# Restore from backup
docker exec -i krakenhashes-postgres pg_restore \
  -U krakenhashes \
  -d krakenhashes \
  --verbose \
  < /backup/krakenhashes/postgres/20240101/krakenhashes_20240101_120000.dump

# Start all services
docker-compose up -d
```

### 2. File System Restore

```bash
# Create data directory if it doesn't exist
mkdir -p /var/lib/krakenhashes

# Extract backup
tar -xzf /backup/krakenhashes/files/20240101/krakenhashes_data_20240101_120000.tar.gz \
  -C /var/lib/krakenhashes

# Verify checksum
cd /backup/krakenhashes/files/20240101
sha256sum -c krakenhashes_data_20240101_120000.tar.gz.sha256

# Fix permissions
chown -R 1000:1000 /var/lib/krakenhashes
chmod -R 750 /var/lib/krakenhashes
```

### 3. Docker Volume Restore

```bash
# Stop services
docker-compose down

# Remove existing volumes
docker volume rm krakenhashes_postgres_data krakenhashes_app_data

# Recreate volumes
docker volume create krakenhashes_postgres_data
docker volume create krakenhashes_app_data

# Restore PostgreSQL volume
docker run --rm \
  -v krakenhashes_postgres_data:/target \
  -v /backup/krakenhashes/volumes/20240101:/backup:ro \
  alpine tar -xzf /backup/postgres_data_20240101_120000.tar.gz -C /target

# Restore application data volume
docker run --rm \
  -v krakenhashes_app_data:/target \
  -v /backup/krakenhashes/volumes/20240101:/backup:ro \
  alpine tar -xzf /backup/app_data_20240101_120000.tar.gz -C /target

# Start services
docker-compose up -d
```

## Backup Verification

### Automated Verification Script

Create `/usr/local/bin/krakenhashes-verify-backup.sh`:

```bash
#!/bin/bash

# KrakenHashes Backup Verification Script
# Verifies backup integrity and tests restore procedures

set -e

# Configuration
BACKUP_DIR="/backup/krakenhashes"
TEST_RESTORE_DIR="/tmp/krakenhashes-restore-test"
VERIFICATION_LOG="/var/log/krakenhashes-backup-verification.log"

# Function to log messages
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "${VERIFICATION_LOG}"
}

# Function to verify file integrity
verify_file() {
    local file=$1
    local checksum_file="${file}.sha256"
    
    if [ -f "${checksum_file}" ]; then
        if sha256sum -c "${checksum_file}" > /dev/null 2>&1; then
            log "✓ Checksum verified: $(basename ${file})"
            return 0
        else
            log "✗ Checksum failed: $(basename ${file})"
            return 1
        fi
    else
        log "⚠ No checksum file for: $(basename ${file})"
        return 2
    fi
}

# Function to test database restore
test_db_restore() {
    local backup_file=$1
    
    log "Testing database restore from: $(basename ${backup_file})"
    
    # Create test database
    docker exec krakenhashes-postgres psql -U postgres -c "CREATE DATABASE krakenhashes_test;"
    
    # Attempt restore
    if gunzip -c "${backup_file}" | docker exec -i krakenhashes-postgres psql -U postgres -d krakenhashes_test > /dev/null 2>&1; then
        # Verify some data
        USERS=$(docker exec krakenhashes-postgres psql -U postgres -d krakenhashes_test -t -c "SELECT COUNT(*) FROM users;")
        log "✓ Database restore successful. Found ${USERS// /} users."
        
        # Clean up
        docker exec krakenhashes-postgres psql -U postgres -c "DROP DATABASE krakenhashes_test;"
        return 0
    else
        log "✗ Database restore failed"
        docker exec krakenhashes-postgres psql -U postgres -c "DROP DATABASE IF EXISTS krakenhashes_test;"
        return 1
    fi
}

# Main verification process
log "Starting backup verification..."

# Find latest backups
LATEST_DB_BACKUP=$(find "${BACKUP_DIR}/postgres" -name "*.dump.gz" -type f -mtime -1 | sort -r | head -1)
LATEST_FILE_BACKUP=$(find "${BACKUP_DIR}/files" -name "*.tar.gz" -type f -mtime -1 | sort -r | head -1)
LATEST_VOLUME_BACKUP=$(find "${BACKUP_DIR}/volumes" -name "postgres_data_*.tar.gz" -type f -mtime -1 | sort -r | head -1)

# Verify database backup
if [ -n "${LATEST_DB_BACKUP}" ]; then
    verify_file "${LATEST_DB_BACKUP}"
    test_db_restore "${LATEST_DB_BACKUP}"
else
    log "⚠ No recent database backup found"
fi

# Verify file backup
if [ -n "${LATEST_FILE_BACKUP}" ]; then
    verify_file "${LATEST_FILE_BACKUP}"
    
    # Test extraction
    mkdir -p "${TEST_RESTORE_DIR}"
    if tar -tzf "${LATEST_FILE_BACKUP}" > /dev/null 2>&1; then
        log "✓ File backup archive is valid"
    else
        log "✗ File backup archive is corrupted"
    fi
    rm -rf "${TEST_RESTORE_DIR}"
else
    log "⚠ No recent file backup found"
fi

# Verify volume backup
if [ -n "${LATEST_VOLUME_BACKUP}" ]; then
    verify_file "${LATEST_VOLUME_BACKUP}"
else
    log "⚠ No recent volume backup found"
fi

log "Backup verification completed"
```

## Disaster Recovery Plan

### Recovery Time Objectives (RTO)

- **Database**: 30 minutes
- **File System**: 1 hour
- **Full System**: 2 hours

### Recovery Point Objectives (RPO)

- **Database**: 6 hours (based on backup frequency)
- **File System**: 24 hours
- **Configuration**: 7 days

### Recovery Priority

1. **PostgreSQL Database** - Contains all system state
2. **Configuration Files** - Required for system operation
3. **Hashlists** - User-uploaded hash files
4. **Wordlists/Rules** - Can be re-downloaded if needed
5. **Binaries** - Can be re-downloaded from version tracking

### Emergency Recovery Checklist

1. **Assess Damage**
   - [ ] Identify failed components
   - [ ] Determine data loss extent
   - [ ] Document incident timeline

2. **Prepare Recovery Environment**
   - [ ] Provision new hardware/VMs
   - [ ] Install Docker and dependencies
   - [ ] Restore network configuration

3. **Restore Core Services**
   - [ ] Restore PostgreSQL database
   - [ ] Restore configuration files
   - [ ] Verify TLS certificates

4. **Restore Data**
   - [ ] Restore file system data
   - [ ] Verify hashlist integrity
   - [ ] Restore Docker volumes

5. **Validation**
   - [ ] Test user authentication
   - [ ] Verify agent connectivity
   - [ ] Check job execution
   - [ ] Validate data integrity

6. **Communication**
   - [ ] Notify users of recovery status
   - [ ] Document lessons learned
   - [ ] Update recovery procedures

## Best Practices

1. **Regular Testing**
   - Test restore procedures monthly
   - Perform full disaster recovery drill quarterly
   - Document test results and issues

2. **Off-site Storage**
   - Keep backups in multiple locations
   - Use cloud storage for critical backups
   - Maintain 3-2-1 backup strategy (3 copies, 2 different media, 1 off-site)

3. **Monitoring**
   - Monitor backup job success/failure
   - Alert on backup size anomalies
   - Track backup storage usage

4. **Security**
   - Encrypt backups at rest
   - Restrict backup access
   - Regularly rotate backup credentials

5. **Documentation**
   - Keep recovery procedures updated
   - Document system dependencies
   - Maintain contact information for key personnel

## Troubleshooting

### Common Issues

1. **Backup Fails with "Permission Denied"**
   ```bash
   # Fix backup directory permissions
   sudo chown -R $(whoami):$(whoami) /backup/krakenhashes
   sudo chmod -R 750 /backup/krakenhashes
   ```

2. **Database Restore Fails**
   ```bash
   # Check PostgreSQL logs
   docker logs krakenhashes-postgres
   
   # Verify database exists
   docker exec krakenhashes-postgres psql -U postgres -l
   
   # Check user permissions
   docker exec krakenhashes-postgres psql -U postgres -c "\du"
   ```

3. **Insufficient Disk Space**
   ```bash
   # Check disk usage
   df -h /backup
   
   # Clean old backups manually
   find /backup/krakenhashes -type f -mtime +${DAYS} -delete
   ```

4. **Slow Backup Performance**
   ```bash
   # Use parallel compression
   tar -cf - -C /var/lib/krakenhashes . | pigz > backup.tar.gz
   
   # Adjust PostgreSQL backup parameters
   docker exec krakenhashes-postgres psql -U postgres -c "SET maintenance_work_mem = '1GB';"
   ```

## Monitoring and Alerting

### Backup Monitoring Script

Create `/usr/local/bin/krakenhashes-backup-monitor.sh`:

```bash
#!/bin/bash

# Check if backups are current and send alerts

BACKUP_DIR="/backup/krakenhashes"
MAX_AGE_HOURS=25  # Alert if backup is older than 25 hours
ALERT_EMAIL="admin@example.com"

check_backup_age() {
    local backup_type=$1
    local pattern=$2
    
    latest=$(find "${BACKUP_DIR}/${backup_type}" -name "${pattern}" -type f -mtime -1 | sort -r | head -1)
    
    if [ -z "$latest" ]; then
        echo "CRITICAL: No recent ${backup_type} backup found" | \
          mail -s "KrakenHashes Backup Alert" "${ALERT_EMAIL}"
    fi
}

# Check each backup type
check_backup_age "postgres" "*.dump.gz"
check_backup_age "files" "*.tar.gz"
check_backup_age "volumes" "postgres_data_*.tar.gz"
```

Add to crontab:
```cron
# Monitor backups daily at 9 AM
0 9 * * * /usr/local/bin/krakenhashes-backup-monitor.sh
```

## Summary

This backup strategy ensures:

- **Comprehensive Coverage**: All critical data is backed up
- **Automation**: Reduces human error and ensures consistency
- **Verification**: Regular testing of backup integrity
- **Quick Recovery**: Clear procedures for various failure scenarios
- **Scalability**: Procedures scale with system growth

Regular review and testing of these procedures is essential for maintaining a reliable backup and recovery system.