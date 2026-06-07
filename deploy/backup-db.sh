#!/bin/bash
# Backup PostgreSQL (Docker) to /opt/lao-rice-api/backups/postgres/
# Safe to run manually or from cron. Does not stop the API.
set -euo pipefail

APP_DIR="/opt/lao-rice-api"
BACKUP_DIR="$APP_DIR/backups/postgres"
KEEP_COUNT="${KEEP_COUNT:-14}"

dc() {
  if docker compose version >/dev/null 2>&1; then
    docker compose -f "$APP_DIR/docker-compose.yml" "$@"
  else
    docker-compose -f "$APP_DIR/docker-compose.yml" "$@"
  fi
}

cd "$APP_DIR"
mkdir -p "$BACKUP_DIR"

for i in $(seq 1 30); do
  if dc exec -T db pg_isready -U postgres -d lao_rice >/dev/null 2>&1; then
    break
  fi
  if [ "$i" -eq 30 ]; then
    echo "ERROR: PostgreSQL not ready" >&2
    exit 1
  fi
  sleep 1
done

STAMP="$(date +%Y%m%d_%H%M%S)"
OUT="$BACKUP_DIR/lao_rice_${STAMP}.sql.gz"

if dc exec -T db pg_dump -U postgres --no-owner --no-acl lao_rice | gzip >"$OUT"; then
  if [ ! -s "$OUT" ]; then
    rm -f "$OUT"
    echo "ERROR: backup file is empty" >&2
    exit 1
  fi
  echo "Backup OK: $OUT"
  ls -1t "$BACKUP_DIR"/lao_rice_*.sql.gz 2>/dev/null | tail -n +$((KEEP_COUNT + 1)) | xargs -r rm -f
else
  rm -f "$OUT"
  echo "ERROR: pg_dump failed" >&2
  exit 1
fi
