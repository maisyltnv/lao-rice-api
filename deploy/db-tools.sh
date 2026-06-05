#!/bin/bash
# Shared DB helpers for deploy scripts (source, do not execute directly).

APP_DIR="${APP_DIR:-/opt/lao-rice-api}"
DB_PASSWORD=""

dc() {
  if docker compose version >/dev/null 2>&1; then
    docker compose -f "$APP_DIR/docker-compose.yml" "$@"
  else
    docker-compose -f "$APP_DIR/docker-compose.yml" "$@"
  fi
}

db_password_from_url() {
  echo "$1" | sed -n 's|.*postgres://postgres:\([^@]*\)@.*|\1|p'
}

# Resolve DB password, persist .db_password if missing. Sets DB_PASSWORD or exits.
ensure_db_password() {
  local backup_file="${1:-}"
  DB_PASSWORD=""

  if [ -f "$APP_DIR/.db_password" ]; then
    DB_PASSWORD="$(cat "$APP_DIR/.db_password")"
    [ -n "$DB_PASSWORD" ] || { echo "ERROR: .db_password is empty" >&2; exit 1; }
    return 0
  fi

  local url=""
  if [ -n "$backup_file" ] && [ -s "$backup_file" ]; then
    url="$(grep -E '^DATABASE_URL=' "$backup_file" | head -1 | cut -d= -f2- || true)"
    DB_PASSWORD="$(db_password_from_url "$url")"
  fi
  if [ -z "$DB_PASSWORD" ] && [ -f "$APP_DIR/.env" ]; then
    url="$(grep -E '^DATABASE_URL=' "$APP_DIR/.env" | head -1 | cut -d= -f2- || true)"
    DB_PASSWORD="$(db_password_from_url "$url")"
  fi

  if [ -z "$DB_PASSWORD" ]; then
    echo "ERROR: No .db_password and no DATABASE_URL in .env backup." >&2
    echo "Fix: echo -n 'YOUR_DB_PASSWORD' > $APP_DIR/.db_password && chmod 600 $APP_DIR/.db_password" >&2
    exit 1
  fi

  printf '%s' "$DB_PASSWORD" > "$APP_DIR/.db_password"
  chmod 600 "$APP_DIR/.db_password"
  echo "==> Saved missing .db_password from existing config"
}

write_production_env() {
  local api_port="${1:-8081}"
  local jwt_secret="${2:-change-me-in-production-use-long-random-secret}"
  cat > "$APP_DIR/.env" <<EOF
PORT=${api_port}
DATABASE_URL=postgres://postgres:${DB_PASSWORD}@localhost:5433/lao_rice?sslmode=disable
JWT_SECRET=${jwt_secret}
JWT_EXPIRY_HOURS=72
SHIPPING_FEE_LAK=30000
FREE_SHIPPING_MIN_SUBTOTAL_LAK=500000
UPLOAD_DIR=${APP_DIR}/uploads
UPLOAD_URL_PREFIX=/uploads
IMAGES_DIR=${APP_DIR}/images
EOF
}

sync_compose_postgres_password() {
  if [ -f "$APP_DIR/docker-compose.yml" ]; then
    sed -i "s/POSTGRES_PASSWORD: .*/POSTGRES_PASSWORD: ${DB_PASSWORD}/" "$APP_DIR/docker-compose.yml"
  fi
}

wait_for_postgres() {
  local max="${1:-30}"
  for _ in $(seq 1 "$max"); do
    if dc exec -T db pg_isready -U postgres -d lao_rice >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  echo "ERROR: PostgreSQL not ready after ${max}s" >&2
  return 1
}

verify_db_login() {
  dc exec -T -e PGPASSWORD="$DB_PASSWORD" db psql -h 127.0.0.1 -U postgres -d lao_rice -c 'SELECT 1' >/dev/null 2>&1
}

sync_postgres_password() {
  local escaped
  escaped="$(printf '%s' "$DB_PASSWORD" | sed "s/'/''/g")"
  echo "==> Syncing PostgreSQL password to match .db_password (data preserved)"
  dc exec -T db psql -U postgres -d lao_rice -c "ALTER USER postgres PASSWORD '${escaped}';"
}

ensure_db_login() {
  if verify_db_login; then
    echo "==> Database login verified"
    return 0
  fi
  sync_postgres_password
  if verify_db_login; then
    echo "==> Database login OK after password sync"
    return 0
  fi
  echo "ERROR: Cannot connect to PostgreSQL with password from .db_password" >&2
  echo "Check: docker compose ps && cat $APP_DIR/.db_password" >&2
  exit 1
}

backup_database() {
  local dir="$APP_DIR/backups/postgres"
  mkdir -p "$dir"
  local file="$dir/pre_deploy_$(date +%Y%m%d_%H%M%S).sql"
  if dc exec -T db pg_dump -U postgres lao_rice >"$file" 2>/dev/null && [ -s "$file" ]; then
    echo "==> DB backup saved: $file"
    ls -1t "$dir"/pre_deploy_*.sql 2>/dev/null | tail -n +8 | xargs -r rm -f
  else
    rm -f "$file"
    echo "==> DB backup skipped (empty database or pg_dump unavailable)"
  fi
}
