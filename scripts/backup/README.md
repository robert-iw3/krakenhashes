# KrakenHashes Backup Scripts

This directory contains automated backup scripts for KrakenHashes. These scripts are designed to be run via cron for regular, automated backups of your KrakenHashes installation.

## Scripts Overview

1. **krakenhashes-db-backup.sh** - PostgreSQL database backup
2. **krakenhashes-file-backup.sh** - File system backup (binaries, wordlists, rules, hashlists)
3. **krakenhashes-volume-backup.sh** - Docker volume backup
4. **krakenhashes-verify-backup.sh** - Backup verification and integrity checking

## Installation

1. Copy scripts to system location:
```bash
sudo cp *.sh /usr/local/bin/
sudo chmod +x /usr/local/bin/krakenhashes-*.sh
```

2. Create backup directories:
```bash
sudo mkdir -p /backup/krakenhashes/{postgres,files,volumes,config}
sudo chown -R $(whoami):$(whoami) /backup/krakenhashes
```

3. Configure environment variables (if needed):
```bash
export KH_DATA_DIR=/var/lib/krakenhashes
```

4. Add to crontab:
```bash
crontab -e
```

Add the following lines:
```cron
# KrakenHashes Automated Backups
0 */6 * * * /usr/local/bin/krakenhashes-db-backup.sh >> /var/log/krakenhashes-backup.log 2>&1
0 2 * * * /usr/local/bin/krakenhashes-file-backup.sh >> /var/log/krakenhashes-backup.log 2>&1
0 3 * * * /usr/local/bin/krakenhashes-volume-backup.sh >> /var/log/krakenhashes-backup.log 2>&1
0 9 * * * /usr/local/bin/krakenhashes-verify-backup.sh
```

## Configuration

Edit the configuration section in each script to match your environment:

- `BACKUP_DIR` - Where to store backups
- `RETENTION_DAYS` - How long to keep old backups
- `DB_CONTAINER` - PostgreSQL container name
- `DATA_DIR` - KrakenHashes data directory

## Manual Execution

Run any script manually:
```bash
sudo /usr/local/bin/krakenhashes-db-backup.sh
```

## Monitoring

Check backup logs:
```bash
tail -f /var/log/krakenhashes-backup.log
tail -f /var/log/krakenhashes-backup-verification.log
```

## Restore Procedures

See the full documentation at `/docs/admin-guide/operations/backup.md` for detailed restore procedures.