#!/bin/bash
# Restore lao_rice database from a .sql.gz backup created by backup-db.sh
# Usage: bash deploy/restore-db.sh /opt/lao-rice-api/backups/postgres/lao_rice_YYYYMMDD_HHMMSS.sql.gz
set -euo pipefail

APP_DIR="/opt/lao-rice-api"
FILE="${1:-}"

if [ -z "$FILE" ] || [ ! -f "$FILE" ]; then
  echo "Usage: $0 /path/to/lao_rice_YYYYMMDD_HHMMSS.sql.gz" >&2
  echo "List backups: ls -lt $APP_DIR/backups/postgres/" >&2
  exit 1
fi

dc() {
  if docker compose version >/dev/null 2>&1; then
    docker compose -f "$APP_DIR/docker-compose.yml" "$@"
  else
    docker-compose -f "$APP_DIR/docker-compose.yml" "$@"
  fi
}

echo "WARNING: This will REPLACE all current data in database lao_rice."
echo "Backup file: $FILE"
printf "Type YES to continue: "
read -r confirm
if [ "$confirm" != "YES" ]; then
  echo "Aborted."
  exit 1
fi

cd "$APP_DIR"

echo "==> Stop API (avoid open connections)"
sudo systemctl stop lao-rice-api 2>/dev/null || true

echo "==> Restore database"
gzip -dc "$FILE" | dc exec -T db psql -U postgres -d lao_rice -v ON_ERROR_STOP=1

echo "==> Start API"
sudo systemctl start lao-rice-api

echo "Restore complete."
