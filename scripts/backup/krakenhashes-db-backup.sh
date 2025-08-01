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