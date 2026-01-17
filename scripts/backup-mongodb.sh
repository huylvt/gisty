#!/bin/bash
# ===========================================
# MongoDB Backup Script for Gisty
# ===========================================
# This script creates a backup of MongoDB and uploads it to S3
# Run via cron: 0 2 * * * /srv/gisty/scripts/backup-mongodb.sh

set -euo pipefail

# Configuration (override via environment variables)
BACKUP_DIR="${BACKUP_DIR:-/tmp/mongodb-backups}"
MONGO_HOST="${MONGO_HOST:-mongodb}"
MONGO_PORT="${MONGO_PORT:-27017}"
MONGO_DB="${MONGO_DB:-gisty}"
MONGO_USER="${MONGO_USER:-gisty}"
MONGO_PASSWORD="${MONGO_PASSWORD:-}"
S3_BUCKET="${S3_BACKUP_BUCKET:-gisty-backups}"
S3_ENDPOINT="${S3_ENDPOINT:-}"
RETENTION_DAYS="${RETENTION_DAYS:-7}"

# Generate timestamp
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_NAME="gisty_${TIMESTAMP}"
BACKUP_PATH="${BACKUP_DIR}/${BACKUP_NAME}"

# Logging function
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

# Error handling
error_exit() {
    log "ERROR: $1"
    exit 1
}

# Create backup directory
mkdir -p "${BACKUP_DIR}"

log "Starting MongoDB backup..."

# Build mongodump command
MONGODUMP_CMD="mongodump --host ${MONGO_HOST} --port ${MONGO_PORT} --db ${MONGO_DB} --out ${BACKUP_PATH}"

if [ -n "${MONGO_USER}" ]; then
    MONGODUMP_CMD="${MONGODUMP_CMD} --username ${MONGO_USER} --authenticationDatabase admin"
fi

if [ -n "${MONGO_PASSWORD}" ]; then
    MONGODUMP_CMD="${MONGODUMP_CMD} --password ${MONGO_PASSWORD}"
fi

# Execute backup
log "Running mongodump..."
eval "${MONGODUMP_CMD}" || error_exit "mongodump failed"

# Compress backup
log "Compressing backup..."
cd "${BACKUP_DIR}"
tar -czf "${BACKUP_NAME}.tar.gz" "${BACKUP_NAME}" || error_exit "Compression failed"
rm -rf "${BACKUP_NAME}"

BACKUP_FILE="${BACKUP_DIR}/${BACKUP_NAME}.tar.gz"
BACKUP_SIZE=$(du -h "${BACKUP_FILE}" | cut -f1)
log "Backup created: ${BACKUP_FILE} (${BACKUP_SIZE})"

# Upload to S3
log "Uploading to S3..."
S3_CMD="aws s3 cp ${BACKUP_FILE} s3://${S3_BUCKET}/mongodb/${BACKUP_NAME}.tar.gz"

if [ -n "${S3_ENDPOINT}" ]; then
    S3_CMD="${S3_CMD} --endpoint-url ${S3_ENDPOINT}"
fi

eval "${S3_CMD}" || error_exit "S3 upload failed"
log "Backup uploaded to s3://${S3_BUCKET}/mongodb/${BACKUP_NAME}.tar.gz"

# Cleanup local backup
rm -f "${BACKUP_FILE}"
log "Local backup file removed"

# Remove old backups from S3 (retention policy)
log "Cleaning up old backups (retention: ${RETENTION_DAYS} days)..."
CUTOFF_DATE=$(date -d "-${RETENTION_DAYS} days" +%Y%m%d)

S3_LIST_CMD="aws s3 ls s3://${S3_BUCKET}/mongodb/"
if [ -n "${S3_ENDPOINT}" ]; then
    S3_LIST_CMD="${S3_LIST_CMD} --endpoint-url ${S3_ENDPOINT}"
fi

eval "${S3_LIST_CMD}" | while read -r line; do
    FILE_NAME=$(echo "$line" | awk '{print $4}')
    if [ -n "${FILE_NAME}" ]; then
        # Extract date from filename (gisty_YYYYMMDD_HHMMSS.tar.gz)
        FILE_DATE=$(echo "${FILE_NAME}" | grep -oP 'gisty_\K\d{8}' || true)
        if [ -n "${FILE_DATE}" ] && [ "${FILE_DATE}" -lt "${CUTOFF_DATE}" ]; then
            log "Removing old backup: ${FILE_NAME}"
            S3_RM_CMD="aws s3 rm s3://${S3_BUCKET}/mongodb/${FILE_NAME}"
            if [ -n "${S3_ENDPOINT}" ]; then
                S3_RM_CMD="${S3_RM_CMD} --endpoint-url ${S3_ENDPOINT}"
            fi
            eval "${S3_RM_CMD}"
        fi
    fi
done

log "Backup completed successfully!"
