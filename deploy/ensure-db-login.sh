#!/bin/bash
# Keep PostgreSQL password in sync with .db_password (safe to run from cron).
set -euo pipefail

APP_DIR="/opt/lao-rice-api"

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

if [ ! -f .db_password ]; then
  echo "WARN: $APP_DIR/.db_password not found — skip DB login check" >&2
  exit 0
fi

DB_PASSWORD="$(tr -d '\n' < .db_password)"
if [ -z "$DB_PASSWORD" ]; then
  echo "WARN: .db_password is empty" >&2
  exit 0
fi

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

verify_login() {
  dc exec -T -e PGPASSWORD="$DB_PASSWORD" db psql -h 127.0.0.1 -U postgres -d lao_rice -c 'SELECT 1' >/dev/null 2>&1
}

if verify_login; then
  echo "DB login OK"
else
  echo "DB password mismatch — syncing from .db_password"
  escaped="$(printf '%s' "$DB_PASSWORD" | sed "s/'/''/g")"
  dc exec -T db psql -U postgres -d lao_rice -c "ALTER USER postgres PASSWORD '${escaped}';"
  if ! verify_login; then
    echo "ERROR: DB login still failing after sync" >&2
    exit 1
  fi
  echo "DB login OK after sync"
fi

API_PORT="${PORT:-8081}"
if [ -f "$APP_DIR/.env" ]; then
  # shellcheck disable=SC1091
  source "$APP_DIR/.env"
  API_PORT="${PORT:-8081}"
fi

if curl -sf "http://127.0.0.1:${API_PORT}/health" >/dev/null 2>&1; then
  echo "API health OK"
  exit 0
fi

echo "API unhealthy — restarting lao-rice-api"
if command -v systemctl >/dev/null 2>&1; then
  sudo systemctl restart lao-rice-api 2>/dev/null || true
  sleep 3
fi

if curl -sf "http://127.0.0.1:${API_PORT}/health" >/dev/null 2>&1; then
  echo "API health OK after restart"
  exit 0
fi

echo "ERROR: API still unhealthy after restart" >&2
exit 1
