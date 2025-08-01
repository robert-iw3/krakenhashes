# Storage Architecture

## Overview

KrakenHashes implements a centralized file storage system with intelligent deduplication, hash verification, and performance optimizations. This guide covers the storage architecture, capacity planning, and maintenance procedures.

## Storage Directory Structure

The system organizes files into a hierarchical structure under the configured data directory (default: `/var/lib/krakenhashes` in Docker, `~/.krakenhashes-data` locally):

```
/var/lib/krakenhashes/           # Root data directory (KH_DATA_DIR)
├── binaries/                    # Hashcat/John binaries
│   ├── hashcat_7.6.0_linux64.tar.gz
│   └── john_1.9.0_linux64.tar.gz
├── wordlists/                   # Wordlist files by category
│   ├── general/                 # Common wordlists
│   ├── specialized/             # Domain-specific lists
│   ├── targeted/                # Custom targeted lists
│   └── custom/                  # User-uploaded lists
├── rules/                       # Rule files by type
│   ├── hashcat/                 # Hashcat-compatible rules
│   ├── john/                    # John-compatible rules
│   └── custom/                  # Custom rule sets
├── hashlists/                   # Processed hashlist files
│   ├── 1.hash                   # Uncracked hashes for job ID 1
│   └── 2.hash                   # Uncracked hashes for job ID 2
├── hashlist_uploads/            # Temporary upload storage
│   └── <user-id>/              # User-specific upload directories
└── local/                       # Extracted binaries (server-side)
```

### Directory Permissions

All directories are created with mode `0750` (rwxr-x---) to ensure:
- Owner has full access
- Group has read and execute access
- Others have no access

## File Deduplication and Hash Verification

### MD5-Based Deduplication

KrakenHashes uses MD5 hashes for file deduplication across all resource types:

1. **Upload Processing**
   - Calculate MD5 hash of uploaded file
   - Check database for existing file with same hash
   - If exists, reference existing file instead of storing duplicate
   - If new, store file and record hash in database

2. **Verification States**
   - `pending` - File uploaded but not yet verified
   - `verified` - File hash matches database record
   - `failed` - Hash mismatch or file corrupted
   - `deleted` - File removed from storage

3. **Database Schema**
   ```sql
   -- Example: Wordlists table
   CREATE TABLE wordlists (
       id SERIAL PRIMARY KEY,
       name VARCHAR(255) NOT NULL,
       file_name VARCHAR(255) NOT NULL,
       md5_hash VARCHAR(32) NOT NULL,
       file_size BIGINT NOT NULL,
       verification_status VARCHAR(20) DEFAULT 'pending',
       UNIQUE(md5_hash)  -- Ensures deduplication
   );
   ```

### File Synchronization

The agent file sync system ensures consistency across distributed agents:

1. **Sync Protocol**
   - Agent reports current files with MD5 hashes
   - Server compares against master file list
   - Server sends list of files to download
   - Agent downloads only missing/changed files

2. **Hash Verification**
   - Files are verified after download
   - Failed verifications trigger re-download
   - Corrupted files are automatically replaced

## Storage Requirements and Capacity Planning

### Estimating Storage Needs

Calculate storage requirements based on:

1. **Wordlists**
   - Common wordlists: 10-50 GB
   - Specialized lists: 50-200 GB
   - Large collections: 500+ GB

2. **Rules**
   - Basic rule sets: 1-10 MB
   - Comprehensive sets: 100-500 MB

3. **Hashlists**
   - Original uploads: Variable
   - Processed files: ~32 bytes per hash
   - Example: 1M hashes ≈ 32 MB

4. **Binaries**
   - Hashcat package: ~100 MB
   - John package: ~50 MB
   - Multiple versions: Plan for 3-5 versions

### Recommended Minimums

| Deployment Size | Storage | Rationale |
|----------------|---------|-----------|
| Development | 50 GB | Basic wordlists and testing |
| Small Team | 200 GB | Standard wordlists + custom data |
| Enterprise | 1 TB+ | Comprehensive wordlists + history |

### Growth Considerations

- **Hashlist accumulation**: ~10-20% monthly growth typical
- **Wordlist expansion**: New lists added periodically
- **Binary versions**: Keep 3-5 recent versions
- **Backup overhead**: 2x storage for full backups

## Backup Considerations

### What to Backup

1. **Critical Data**
   - PostgreSQL database (contains all metadata)
   - Custom wordlists and rules
   - Configuration files (`/etc/krakenhashes`)

2. **Recoverable Data**
   - Standard wordlists (can be re-downloaded)
   - Binaries (can be re-downloaded)
   - Processed hashlists (can be regenerated)

### Backup Strategy

```bash
#!/bin/bash
# Example backup script

# Backup database
pg_dump -h postgres -U krakenhashes krakenhashes > backup/db_$(date +%Y%m%d).sql

# Backup custom data
rsync -av /var/lib/krakenhashes/wordlists/custom/ backup/wordlists/
rsync -av /var/lib/krakenhashes/rules/custom/ backup/rules/
rsync -av /etc/krakenhashes/ backup/config/

# Backup file metadata
docker-compose exec backend \
  psql -c "COPY (SELECT * FROM wordlists) TO STDOUT CSV" > backup/wordlists_meta.csv
```

### Restore Procedures

1. **Database Restore**
   ```bash
   psql -h postgres -U krakenhashes krakenhashes < backup/db_20240115.sql
   ```

2. **File Restore**
   ```bash
   rsync -av backup/wordlists/ /var/lib/krakenhashes/wordlists/custom/
   rsync -av backup/rules/ /var/lib/krakenhashes/rules/custom/
   ```

3. **Verify Integrity**
   - Run file verification for all restored files
   - Check MD5 hashes against database records

## Performance Optimization

### File System Considerations

1. **File System Choice**
   - ext4: Good general performance
   - XFS: Better for large files
   - ZFS: Built-in deduplication and compression

2. **Mount Options**
   ```bash
   # Example /etc/fstab entry with optimizations
   /dev/sdb1 /var/lib/krakenhashes ext4 defaults,noatime,nodiratime 0 2
   ```

3. **Storage Layout**
   - Use separate volumes for different data types
   - Consider SSD for hashlists (frequent reads)
   - HDDs acceptable for wordlists (sequential reads)

### Caching Strategy

1. **Application-Level Caching**
   - Recently used wordlists kept in memory
   - Hash type definitions cached
   - File metadata cached for 15 minutes

2. **File System Caching**
   - Linux page cache handles frequently accessed files
   - Monitor with `free -h` and adjust `vm.vfs_cache_pressure`

### I/O Optimization

```bash
# Tune kernel parameters for better I/O
echo 'vm.dirty_ratio = 5' >> /etc/sysctl.conf
echo 'vm.dirty_background_ratio = 2' >> /etc/sysctl.conf
echo 'vm.vfs_cache_pressure = 50' >> /etc/sysctl.conf
sysctl -p
```

## Docker Volume Management

### Volume Configuration

Docker Compose creates named volumes for persistent storage:

```yaml
volumes:
  krakenhashes_data:        # Main data directory
    name: krakenhashes_app_data
  postgres_data:            # Database storage
    name: krakenhashes_postgres_data
```

### Volume Operations

1. **Inspect Volumes**
   ```bash
   docker volume inspect krakenhashes_app_data
   docker volume ls
   ```

2. **Backup Volumes**
   ```bash
   # Backup data volume
   docker run --rm -v krakenhashes_app_data:/data \
     -v $(pwd)/backup:/backup \
     alpine tar czf /backup/data_backup.tar.gz -C /data .
   ```

3. **Restore Volumes**
   ```bash
   # Restore data volume
   docker run --rm -v krakenhashes_app_data:/data \
     -v $(pwd)/backup:/backup \
     alpine tar xzf /backup/data_backup.tar.gz -C /data
   ```

### Storage Driver Optimization

For production deployments:

```json
{
  "storage-driver": "overlay2",
  "storage-opts": [
    "overlay2.override_kernel_check=true"
  ]
}
```

## File Cleanup and Maintenance

### Automated Cleanup

The system includes automated cleanup for:

1. **Temporary Upload Files**
   - Deleted after successful processing
   - Orphaned files cleaned after 24 hours

2. **Old Hashlist Files**
   - Configurable retention period
   - Default: Keep for job lifetime + 30 days

### Manual Cleanup Procedures

1. **Remove Orphaned Files**
   ```bash
   # Find files not referenced in database
   docker-compose exec backend bash
   cd /var/lib/krakenhashes
   
   # Check for orphaned wordlists
   find wordlists -type f -name "*.txt" | while read f; do
     hash=$(md5sum "$f" | cut -d' ' -f1)
     # Query database for hash
   done
   ```

2. **Clean Old Hashlists**
   ```sql
   -- Remove hashlists older than 90 days with no active jobs
   DELETE FROM hashlists 
   WHERE updated_at < NOW() - INTERVAL '90 days'
   AND id NOT IN (
     SELECT DISTINCT hashlist_id 
     FROM job_executions 
     WHERE status IN ('pending', 'running')
   );
   ```

3. **Vacuum Database**
   ```bash
   docker-compose exec postgres \
     psql -U krakenhashes -c "VACUUM ANALYZE;"
   ```

### Storage Monitoring

1. **Disk Usage Monitoring**
   ```bash
   # Monitor storage usage
   df -h /var/lib/krakenhashes
   du -sh /var/lib/krakenhashes/*
   
   # Set up alerts
   echo '0 * * * * root df -h | grep krakenhashes | \
     awk '\''$5+0 > 80 {print "Storage warning: " $0}'\''' \
     >> /etc/crontab
   ```

2. **File Count Monitoring**
   ```sql
   -- Monitor file counts
   SELECT 
     'wordlists' as type, COUNT(*) as count,
     SUM(file_size)/1024/1024/1024 as size_gb
   FROM wordlists
   WHERE verification_status = 'verified'
   UNION ALL
   SELECT 'rules', COUNT(*), SUM(file_size)/1024/1024/1024
   FROM rules
   WHERE verification_status = 'verified';
   ```

### Best Practices

1. **Regular Maintenance Schedule**
   - Weekly: Check disk usage and clean temp files
   - Monthly: Verify file integrity and clean old hashlists
   - Quarterly: Full backup and storage audit

2. **Monitoring Alerts**
   - Set up alerts for >80% disk usage
   - Monitor file verification failures
   - Track deduplication efficiency

3. **Documentation**
   - Document custom wordlist sources
   - Maintain changelog for rule modifications
   - Record storage growth trends

## Troubleshooting

### Common Issues

1. **Disk Space Exhaustion**
   ```bash
   # Emergency cleanup
   find /var/lib/krakenhashes/hashlist_uploads -mtime +1 -delete
   docker system prune -f
   ```

2. **File Verification Failures**
   ```sql
   -- Find failed verifications
   SELECT * FROM wordlists 
   WHERE verification_status = 'failed';
   
   -- Reset for re-verification
   UPDATE wordlists 
   SET verification_status = 'pending' 
   WHERE verification_status = 'failed';
   ```

3. **Permission Issues**
   ```bash
   # Fix permissions
   chown -R 1000:1000 /var/lib/krakenhashes
   chmod -R 750 /var/lib/krakenhashes
   ```

### Debug Commands

```bash
# Check file system integrity
fsck -n /dev/sdb1

# Monitor I/O performance
iostat -x 1

# Check open files
lsof | grep krakenhashes

# Verify Docker volumes
docker volume inspect krakenhashes_app_data
```