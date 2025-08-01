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