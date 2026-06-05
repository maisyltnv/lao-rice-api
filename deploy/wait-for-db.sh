#!/bin/bash
set -euo pipefail

APP_DIR="/opt/lao-rice-api"
MAX_WAIT="${MAX_WAIT:-60}"

dc() {
  if docker compose version >/dev/null 2>&1; then
    docker compose -f "$APP_DIR/docker-compose.yml" "$@"
  else
    docker-compose -f "$APP_DIR/docker-compose.yml" "$@"
  fi
}

cd "$APP_DIR"

if ! dc ps --status running --services 2>/dev/null | grep -qx db; then
  echo "PostgreSQL container not running — starting db..."
  dc up -d db
fi

for i in $(seq 1 "$MAX_WAIT"); do
  if dc exec -T db pg_isready -U postgres -d lao_rice >/dev/null 2>&1; then
    exit 0
  fi
  sleep 1
done

echo "PostgreSQL not ready after ${MAX_WAIT}s" >&2
exit 1
