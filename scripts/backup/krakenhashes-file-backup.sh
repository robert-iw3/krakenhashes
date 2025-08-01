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