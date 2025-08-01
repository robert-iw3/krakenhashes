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